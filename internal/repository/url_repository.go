package repository

import (
	"context"
	"database/sql"

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
	// TODO: Implement INSERT
	return nil
}

func (r *URLRepository) FindByShortCode(ctx context.Context, shortCode string) (*models.URL, error) {
	query := `
        SELECT id, short_code, long_url, created_at, expires_at
        FROM urls
        WHERE short_code = $1 AND (expires_at IS NULL OR expires_at > NOW())
    `
	// TODO: Implement SELECT
	return nil, nil
}

func (r *URLRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`
	// TODO: Implement EXISTS check
	return false, nil
}
