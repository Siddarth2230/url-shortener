package service

import (
	"context"
	"errors"

	"github.com/Siddarth2230/url-shortener/internal/models"
	"github.com/Siddarth2230/url-shortener/internal/repository"
	"github.com/Siddarth2230/url-shortener/pkg/idgen"
)

var (
	ErrInvalidURL      = errors.New("invalid URL")
	ErrCustomCodeTaken = errors.New("custom short code already taken")
)

type URLService struct {
	repo      *repository.URLRepository
	generator idgen.Generator // Interface for flexibility
}

func (s *URLService) ShortenURL(ctx context.Context, req models.ShortenRequest) (*models.ShortenResponse, error) {
	// TODO: Implement shortening logic
	// 1. Validate URL (use regex or url.Parse)
	// 2. If custom code provided, check if available
	// 3. Otherwise, generate short code
	// 4. Handle collision (retry up to 3 times)
	// 5. Save to database
	// 6. Return response

	return nil, nil
}

func (s *URLService) GetLongURL(ctx context.Context, shortCode string) (string, error) {
	// TODO: Implement lookup
	// 1. Query database
	// 2. Check if expired
	// 3. Return long URL

	return "", nil
}
