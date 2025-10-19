package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Siddarth2230/url-shortener/internal/models"
	"github.com/Siddarth2230/url-shortener/internal/repository"
	"github.com/Siddarth2230/url-shortener/pkg/cache"
	"github.com/Siddarth2230/url-shortener/pkg/idgen"
	"github.com/jackc/pgconn"
	"github.com/lib/pq"
)

var (
	ErrInvalidURL      = errors.New("invalid URL")
	ErrCustomCodeTaken = errors.New("custom short code already taken")
	ErrNotFound        = errors.New("short code not found")
	ErrExpired         = errors.New("short URL expired")
	ErrGenExhausted    = errors.New("failed to generate unique short code after retries")
)

// URLService provides URL shortening and lookup.
type URLService struct {
	repo          *repository.URLRepository
	generator     idgen.Generator
	BaseURL       string // set to produce absolute short URLs
	l1Cache       *cache.LRUCache
	l2Cache       *cache.RedisCache
	enableL2Cache bool
}

func NewURLService(repo *repository.URLRepository, gen idgen.Generator, baseURL string, cacheSize int) *URLService {
	return &URLService{
		repo:          repo,
		generator:     gen,
		BaseURL:       baseURL,
		l1Cache:       cache.NewLRUCache(cacheSize),
		l2Cache:       nil,
		enableL2Cache: false,
	}
}

// NewURLServiceWithRedis creates a URL service with both L1 and L2 caches
func NewURLServiceWithRedis(repo *repository.URLRepository, gen idgen.Generator, baseURL string, l1CacheSize int, redisCache *cache.RedisCache) *URLService {
	return &URLService{
		repo:          repo,
		generator:     gen,
		BaseURL:       baseURL,
		l1Cache:       cache.NewLRUCache(l1CacheSize),
		l2Cache:       redisCache,
		enableL2Cache: redisCache != nil,
	}
}

var customCodeRE = regexp.MustCompile(`^[A-Za-z0-9\-\_\.]{4,10}$`)

// ShortenURL creates a short code (or uses custom), persists, and returns the response.
func (s *URLService) ShortenURL(ctx context.Context, req models.ShortenRequest) (*models.ShortenResponse, error) {
	// 0. Validate long URL (uses validateURL helper)
	if err := validateURL(req.URL); err != nil {
		return nil, err
	}

	// If custom code provided, validate and try to save once
	if req.CustomCode != "" {
		if err := validateCustomCode(req.CustomCode); err != nil {
			log.Printf("Invalid custom code %q: %v", req.CustomCode, err)
			return nil, err
		}

		// quick existence check
		ok, err := s.repo.ExistsByShortCode(ctx, req.CustomCode)
		if err != nil {
			log.Printf("Error checking existence of custom code %q: %v", req.CustomCode, err)
			return nil, err
		}
		if ok {
			return nil, ErrCustomCodeTaken
		}

		now := time.Now().UTC()
		u := &models.URL{
			ShortCode: req.CustomCode,
			LongURL:   req.URL,
			CreatedAt: now,
			ExpiresAt: nil,
		}

		if err := s.repo.Save(ctx, u); err != nil {
			if isUniqueConstraintErr(err) {
				return nil, ErrCustomCodeTaken
			}
			log.Printf("Error saving custom code %q: %v", req.CustomCode, err)
			return nil, err
		}

		// Cache immediately after creation (user will likely click soon)
		s.cacheURL(ctx, req.CustomCode, u, 1*time.Hour)

		shortURL := req.CustomCode
		if s.BaseURL != "" {
			shortURL = fmt.Sprintf("%s/%s", s.BaseURL, req.CustomCode)
		}
		return &models.ShortenResponse{
			ShortCode: req.CustomCode,
			ShortURL:  shortURL,
			LongURL:   req.URL,
		}, nil
	}

	// if custom code not provided
	u := &models.URL{
		LongURL:   req.URL,
		ExpiresAt: nil,
	}

	const maxAttempts = 6
	code, gErr := s.generateAndSaveUniqueShortCode(ctx, u, maxAttempts)
	if gErr != nil {
		return nil, gErr
	}

	// Cache the newly created URL
	s.cacheURL(ctx, code, u, 1*time.Hour)

	shortURL := code
	if s.BaseURL != "" {
		shortURL = fmt.Sprintf("%s/%s", s.BaseURL, code)
	}
	return &models.ShortenResponse{
		ShortCode: code,
		ShortURL:  shortURL,
		LongURL:   req.URL,
	}, nil
}

// GetLongURL looks up the long URL for a short code and checks expiry.
func (s *URLService) GetLongURL(ctx context.Context, shortCode string) (string, error) {
	if shortCode == "" {
		return "", ErrNotFound
	}

	// ===== CACHE LAYER (L1) =====
	if cached, ok := s.l1Cache.Get(shortCode); ok {
		// Cache hit!
		if url, ok := cached.(*models.URL); ok {
			// Check expiry
			if url.ExpiresAt != nil && time.Now().UTC().After(*url.ExpiresAt) {
				s.invalidateCache(ctx, shortCode)
				return "", ErrExpired
			}
			return url.LongURL, nil
		}
	}

	// ===== CACHE LAYER (L2) =====
	if s.enableL2Cache && s.l2Cache != nil {
		var cachedURL models.URL
		cacheKey := fmt.Sprintf("url:%s", shortCode)

		err := s.l2Cache.Get(ctx, cacheKey, &cachedURL)
		if err == nil {
			// L2 cache hit

			if cachedURL.ExpiresAt != nil && time.Now().UTC().After(*cachedURL.ExpiresAt) {
				s.invalidateCache(ctx, shortCode)
				return "", ErrExpired
			}

			// Store in L1 for next request to THIS server
			s.l1Cache.Put(shortCode, &cachedURL)

			return cachedURL.LongURL, nil
		}
		if !errors.Is(err, cache.ErrCacheMiss) {
			log.Printf("Redis error for key %s: %v", cacheKey, err)
		}
	}

	// L2 Cache miss - continue to database

	// ============================================================
	// LAYER 3: Query Database (PostgreSQL) - SLOWEST
	// ============================================================

	u, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}
	if u == nil {
		s.cacheNotFound(ctx, shortCode)
		return "", ErrNotFound
	}

	if u.ExpiresAt != nil && time.Now().UTC().After(*u.ExpiresAt) {
		return "", ErrExpired
	}

	ttl := s.calculateCacheTTL(u)

	// Save to cache for next time
	s.cacheURL(ctx, shortCode, u, ttl)

	return u.LongURL, nil
}

func (s *URLService) cacheURL(ctx context.Context, shortCode string, u *models.URL, ttl time.Duration) {
	// L1: In-memory cache (synchronous)
	s.l1Cache.Put(shortCode, u)

	// L2: Redis cache (asynchronous to not block response)
	if s.enableL2Cache && s.l2Cache != nil {
		go func() {
			// Use background context to avoid cancellation
			bgCtx := context.Background()
			cacheKey := fmt.Sprintf("url:%s", shortCode)

			err := s.l2Cache.SetWithTTL(bgCtx, cacheKey, u, ttl)
			if err != nil {
				log.Printf("Failed to cache URL in Redis (key=%s): %v", cacheKey, err)
			}
		}()
	}
}

// cacheNotFound stores a "not found" marker to prevent repeated DB queries
func (s *URLService) cacheNotFound(ctx context.Context, shortCode string) {
	notFoundMarker := &models.URL{
		ShortCode: shortCode,
		LongURL:   "__NOT_FOUND__",
		CreatedAt: time.Now().UTC(),
	}

	s.l1Cache.Put(shortCode, notFoundMarker)

	if s.enableL2Cache && s.l2Cache != nil {
		go func() {
			bgCtx := context.Background()
			cacheKey := fmt.Sprintf("url:%s", shortCode)
			s.l2Cache.SetWithTTL(bgCtx, cacheKey, notFoundMarker, 1*time.Minute)
		}()
	}
}

// invalidateCache removes a URL from both cache layers
func (s *URLService) invalidateCache(ctx context.Context, shortCode string) {
	// L1: Remove from this server's cache
	s.l1Cache.Delete(shortCode)

	// L2: Remove from Redis (affects all servers)
	if s.enableL2Cache && s.l2Cache != nil {
		go func() {
			bgCtx := context.Background()
			cacheKey := fmt.Sprintf("url:%s", shortCode)
			err := s.l2Cache.Delete(bgCtx, cacheKey)
			if err != nil {
				log.Printf("Failed to invalidate cache in Redis (key=%s): %v", cacheKey, err)
			}
		}()
	}
}

func (s *URLService) calculateCacheTTL(u *models.URL) time.Duration {
	defaultTTL := 5 * time.Minute

	// If URL has custom expiry, cache until then (but cap at 1 hour)
	if u.ExpiresAt != nil {
		timeUntilExpiry := time.Until(*u.ExpiresAt)
		if timeUntilExpiry > 0 && timeUntilExpiry < 1*time.Hour {
			return timeUntilExpiry
		}
	}

	if time.Since(u.CreatedAt) < 1*time.Hour {
		return 2 * time.Minute
	}

	return defaultTTL
}

// validateCustomCode enforces allowed chars, length, and reserved blacklist.
func validateCustomCode(code string) error {
	if !customCodeRE.MatchString(code) {
		return fmt.Errorf("custom code must be 4-10 chars and may contain letters, numbers, '-', '_' and '.'")
	}

	// Reserved words (lowercase)
	reserved := map[string]struct{}{
		"admin":  {},
		"api":    {},
		"health": {},
		"www":    {},
		"root":   {},
		"login":  {},
		"status": {},
	}
	l := strings.ToLower(code)
	if _, ok := reserved[l]; ok {
		return fmt.Errorf("custom code %q is reserved", code)
	}

	// Block purely numeric codes to avoid confusion with ID-based systems
	allDigits := true
	for _, r := range code {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return fmt.Errorf("custom code must not be purely numeric")
	}

	return nil
}

// generateAndSaveUniqueShortCode generates a code using the configured generator and saves it to DB.
func (s *URLService) generateAndSaveUniqueShortCode(ctx context.Context, u *models.URL, maxAttempts int) (string, error) {
	for i := 0; i < maxAttempts; i++ {

		code, genErr := s.generator.Generate(ctx)
		if genErr != nil {
			return "", fmt.Errorf("generator failed: %w", genErr)
		}
		if code == "" {
			log.Printf("idgen: generator returned empty code (attempt %d/%d)", i+1, maxAttempts)
			// small sleep/jitter could be added here
			continue
		}

		u.ShortCode = code
		now := time.Now().UTC()
		u.CreatedAt = now

		// rely on DB unique constraint to detect races
		if err := s.repo.Save(ctx, u); err != nil {
			if isUniqueConstraintErr(err) {
				// collision — try again
				log.Printf("Save race detected for code=%s, retrying (attempt %d/%d)", code, i+1, maxAttempts)
				continue
			}
			return "", fmt.Errorf("save failed: %w", err)
		}
		return code, nil
	}

	return "", ErrGenExhausted
}

// validateURL checks that the URL is syntactically valid and uses http/https.
func validateURL(urlStr string) error {
	if urlStr == "" {
		log.Println("URL isn't provided! Please provide URL")
		return ErrInvalidURL
	}
	parsed, err := url.ParseRequestURI(urlStr)
	if err != nil {
		log.Println("Invalide URL. Please check")
		return ErrInvalidURL
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		log.Println("Scheme / Host is empty")
		return ErrInvalidURL
	}
	// Restrict to http(s) for redirect safety
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s", parsed.Scheme)
	}

	return nil
}

// isUniqueConstraintErr tries to heuristically detect unique constraint / duplicate key DB errors.
func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}

	// Case 1: pgx / pgconn driver
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	// Case 2: lib/pq driver
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}

	// Case 3: fallback (not expected, but safe)
	// Some wrapped errors might only expose text
	l := strings.ToLower(err.Error())
	return strings.Contains(l, "duplicate key value violates unique constraint")
}

// TODO: Add cache invalidation on update/delete
func (s *URLService) DeleteShortCode(ctx context.Context, shortCode string) error {
	// Delete from DB
	if err := s.repo.DeleteByShortCode(ctx, shortCode); err != nil {
		return err
	}

	// Invalidate cache
	s.invalidateCache(ctx, shortCode)

	return nil
}
