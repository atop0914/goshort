package model

import (
	"time"
)

// URLRecord represents a shortened URL entry
type URLRecord struct {
	Code        string     `json:"code"`
	OriginalURL string     `json:"original_url"`
	ShortURL    string     `json:"short_url"`
	Clicks      int64      `json:"clicks"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ShortenRequest represents the API request to create a short URL
type ShortenRequest struct {
	URL        string `json:"url"`
	CustomCode string `json:"custom_code,omitempty"`
}

// ShortenResponse represents the API response after creating a short URL
type ShortenResponse struct {
	ShortURL    string    `json:"short_url"`
	Code        string    `json:"code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// URLListResponse represents the response for listing URLs
type URLListResponse struct {
	URLs  []URLRecord `json:"urls"`
	Total int         `json:"total"`
}

// StatsResponse represents the response for URL statistics
type StatsResponse struct {
	Code        string    `json:"code"`
	OriginalURL string    `json:"original_url"`
	ShortURL    string    `json:"short_url"`
	Clicks      int64     `json:"clicks"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
