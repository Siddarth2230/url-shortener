package models

import "time"

type URL struct {
	ID        int64      `json:"id" db:"id"`
	ShortCode string     `json:"short_code" db:"short_code"`
	LongURL   string     `json:"long_url" db:"long_url"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

type ShortenRequest struct {
	URL        string `json:"url" validate:"required,url"`
	CustomCode string `json:"custom_code,omitempty" validate:"omitempty,alphanum,min=4,max=10"`
}

type ShortenResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
	LongURL   string `json:"long_url"`
}
