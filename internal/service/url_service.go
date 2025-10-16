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
	repo      *repository.URLRepository
	generator idgen.Generator
	BaseURL   string // optional; set to produce absolute short URLs
	cache     *cache.LRUCache
}

// NewURLService constructor.
func NewURLService(repo *repository.URLRepository, gen idgen.Generator, baseURL string, cacheSize int) *URLService {
	return &URLService{
		repo:      repo,
		generator: gen,
		BaseURL:   baseURL,
		cache:     cache.NewLRUCache(10),
	}
}

var customCodeRE = regexp.MustCompile(`^[A-Za-z0-9\-\_\.]{4,10}$`)

// ShortenURL creates a short code (or uses custom), persists, and returns the response.
// It retries generation/save on collisions / unique-constraint violations.
func (s *URLService) ShortenURL(ctx context.Context, req models.ShortenRequest) (*models.ShortenResponse, error) {
	// 0. Validate long URL (uses validateURL helper)
	if err := validateURL(req.URL); err != nil {
		return nil, err
	}

	// max attempts for generation/save loops
	const maxAttempts = 6

	// If custom code provided, validate and try to save once (fail if taken).
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
			// If Save failed because of unique constraint (race), treat as taken.
			if isUniqueConstraintErr(err) {
				return nil, ErrCustomCodeTaken
			}
			log.Printf("Error saving custom code %q: %v", req.CustomCode, err)
			return nil, err
		}

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

	// No custom code: generate and try save with retries.
	var lastGenErr error
	for i := 0; i < maxAttempts; i++ {
		code, gErr := s.generateUniqueShortCode(ctx)
		if gErr != nil {
			// generator/database failure is fatal
			lastGenErr = gErr
			break
		}
		if code == "" {
			lastGenErr = errors.New("generator returned empty code")
			continue
		}

		now := time.Now().UTC()
		u := &models.URL{
			ShortCode: code,
			LongURL:   req.URL,
			CreatedAt: now,
			ExpiresAt: nil,
		}

		if err := s.repo.Save(ctx, u); err != nil {
			// race: unique constraint — retry generation loop
			if isUniqueConstraintErr(err) {
				log.Printf("Save race detected for code=%s, retrying (attempt %d/%d)", code, i+1, maxAttempts)
				continue
			}
			// other DB error — bubble up
			return nil, err
		}

		// success
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

	// exhausted attempts or fatal generator error
	if lastGenErr != nil {
		return nil, lastGenErr
	}
	return nil, ErrGenExhausted
}

// GetLongURL looks up the long URL for a short code and checks expiry.
func (s *URLService) GetLongURL(ctx context.Context, shortCode string) (string, error) {
	if shortCode == "" {
		return "", ErrNotFound
	}

	// ===== CACHE LAYER (L1) =====
	if cached, ok := s.cache.Get(shortCode); ok {
		// Cache hit!
		if url, ok := cached.(*models.URL); ok {
			// Check expiry
			if url.ExpiresAt != nil && time.Now().UTC().After(*url.ExpiresAt) {
				s.cache.Put(shortCode, nil) // Invalidate
				return "", ErrExpired
			}
			return url.LongURL, nil
		}
	}

	u, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", ErrNotFound
	}

	if u.ExpiresAt != nil && time.Now().UTC().After(*u.ExpiresAt) {
		return "", ErrExpired
	}

	// Save to cache for next time
	s.cache.Put(shortCode, u)

	return u.LongURL, nil
}

// -------------------- Helpers added per your request --------------------

// validateCustomCode enforces allowed chars, length, and reserved blacklist.
// Returns nil if ok, or an error with a concise reason.
func validateCustomCode(code string) error {
	// Basic length + allowed chars check
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

// generateUniqueShortCode generates a code using the configured generator and checks for existence.
// Retries a few times and logs collisions for monitoring.
func (s *URLService) generateUniqueShortCode(ctx context.Context) (string, error) {
	const maxRetries = 5
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		code, err := s.generator.Generate(ctx)
		if err != nil {
			// generator failure is fatal
			return "", err
		}
		if code == "" {
			lastErr = errors.New("generator returned empty code")
			log.Printf("idgen: empty code on attempt %d", i+1)
			continue
		}

		exists, err := s.repo.ExistsByShortCode(ctx, code)
		if err != nil {
			// DB error
			return "", err
		}
		if !exists {
			// got a unique code
			return code, nil
		}

		// collision: log and retry
		log.Printf("idgen collision detected (attempt=%d code=%s)", i+1, code)
		lastErr = fmt.Errorf("collision on code %s", code)
		// no sleep to keep latency low; could add small backoff if collisions are frequent
	}
	if lastErr != nil {
		return "", ErrGenExhausted
	}
	return "", ErrGenExhausted
}

// validateURL checks that the URL is syntactically valid and uses http/https.
// Tradeoffs:
// - url.ParseRequestURI is cheap and good for basic validation (fast).
// - Requiring http/https prevents weird schemes (mailto:, ftp:).
// - Doing a HEAD/GET to verify existence is slow and can be blocked by remote servers; use cautiously.
// - Blocklist checks require a maintained list and can have false positives.
func validateURL(urlStr string) error {
	if urlStr == "" {
		return ErrInvalidURL
	}
	parsed, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidURL
	}
	// Restrict to http(s) for redirect safety
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s", parsed.Scheme)
	}

	// Optional: existence check (disabled by default). Uncomment if you want to verify remote exists.
	// Warning: this will add latency to shorten requests and may be blocked by some sites.
	/*
		client := http.Client{
			Timeout: 3 * time.Second,
		}
		resp, err := client.Head(urlStr)
		if err != nil || (resp.StatusCode >= 400 && resp.StatusCode != 405) {
			// 405 Method Not Allowed is common — treat as success (resource exists)
			return fmt.Errorf("target URL not reachable: %v", err)
		}
	*/

	// Optional: domain blocklist check — implement as needed.

	return nil
}

// isUniqueConstraintErr tries to heuristically detect unique constraint / duplicate key DB errors.
// Replace with driver-specific checks (e.g., pq, pgconn) for production.
func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	l := strings.ToLower(err.Error())
	return strings.Contains(l, "duplicate") || strings.Contains(l, "unique") || strings.Contains(l, "violates unique constraint")
}

// -------------------- Notes (short and blunt) --------------------
//
// - Retries: I used 5 retries in the generator helper and up to 6 total outer attempts for Save.
//   For a decent generator (>=62^6 space), collisions are extremely unlikely — but keep DB unique constraint and retry on violation.
// - Counter-based generator: will never collide if you map a monotonic ID to Base62. Best for guaranteed uniqueness.
// - Hash/crypto-random: very low collision probability with adequate length; choose length by expected scale.
// - validateURL currently does syntactic checks and forces http/https. Enable HEAD or blocklist checks only if you accept added latency.

// TODO: Add cache invalidation on update/delete
func (s *URLService) DeleteShortCode(ctx context.Context, shortCode string) error {
	// Delete from DB
	if err := s.repo.Delete(ctx, shortCode); err != nil {
		return err
	}

	// Invalidate cache
	s.cache.Put(shortCode, nil)

	return nil
}
