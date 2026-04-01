package model

import "time"

type URLRecord struct {
	Code        string     `json:"code"`
	OriginalURL string     `json:"original_url"`
	ShortURL    string     `json:"short_url"`
	Clicks      int64      `json:"clicks"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type ShortenRequest struct {
	URL        string `json:"url"`
	CustomCode string `json:"custom_code,omitempty"`
}

type ShortenResponse struct {
	ShortURL    string     `json:"short_url"`
	Code        string     `json:"code"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type ListResponse struct {
	URLs  []URLRecord `json:"urls"`
	Total int         `json:"total"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
