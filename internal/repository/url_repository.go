package repository

import (
	"context"
	"database/sql"
	"log"

	"github.com/Siddarth2230/url-shortener/internal/models"
)

type URLRepository struct {
	db *sql.DB
}

func NewURLRepository(db *sql.DB) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Save(ctx context.Context, url *models.URL) error {
	query := `
        INSERT INTO urls (short_code, long_url, created_at, expires_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `
	var expires_at sql.NullTime
	if url.ExpiresAt != nil {
		expires_at = sql.NullTime{Time: *url.ExpiresAt, Valid: true}
	} else {
		expires_at = sql.NullTime{Valid: false}
	}
	row := r.db.QueryRowContext(ctx, query, url.ShortCode, url.LongURL, url.CreatedAt, expires_at)
	if err := row.Scan(&url.ID); err != nil {
		log.Printf("Error saving URL: %v", err)
		return err
	}
	return nil
}

func (r *URLRepository) FindByShortCode(ctx context.Context, shortCode string) (*models.URL, error) {
	query := `
        SELECT id, short_code, long_url, created_at, expires_at
        FROM urls
        WHERE short_code = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`

	var expires_at sql.NullTime
	row := r.db.QueryRowContext(ctx, query, shortCode)
	var url models.URL
	if err := row.Scan(&url.ID, &url.ShortCode, &url.LongURL, &url.CreatedAt, &expires_at); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		log.Printf("Error finding URL by short code: %v", err)
		return nil, err
	}
	if expires_at.Valid {
		url.ExpiresAt = &expires_at.Time
	}
	return &url, nil
}

func (r *URLRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`
	row := r.db.QueryRowContext(ctx, query, shortCode)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		log.Printf("Error checking if URL exists by short code: %v", err)
		return false, err
	}
	return exists, nil
}
