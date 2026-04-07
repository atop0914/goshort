package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/goshort/goshort/internal/model"
	"github.com/goshort/goshort/internal/service"
	"github.com/goshort/goshort/internal/store"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	tokens   map[string]*tokenBucket
	rate     int           // requests per window
	window   time.Duration // time window
	capacity int           // max tokens
}

type tokenBucket struct {
	tokens    int
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, window time.Duration, capacity int) *RateLimiter {
	rl := &RateLimiter{
		tokens:   make(map[string]*tokenBucket),
		rate:     rate,
		window:   window,
		capacity: capacity,
	}
	// Cleanup old entries periodically
	go rl.cleanup()
	return rl
}

// Allow checks if a request from the given key is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.tokens[key]

	if !exists {
		rl.tokens[key] = &tokenBucket{
			tokens:    rl.capacity - 1,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on time passed
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / rl.window) * rl.rate

	if tokensToAdd > 0 {
		bucket.tokens = min(rl.capacity, bucket.tokens+tokensToAdd)
		bucket.lastRefill = now
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// cleanup removes old entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, bucket := range rl.tokens {
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(rl.tokens, key)
			}
		}
		rl.mu.Unlock()
	}
}

// sanitizeURL performs input sanitization on URLs
func sanitizeURL(rawURL string) (string, error) {
	// Trim whitespace
	rawURL = strings.TrimSpace(rawURL)

	// Remove any null bytes
	rawURL = strings.ReplaceAll(rawURL, "\x00", "")

	// Decode URL-encoded characters for validation, then re-encode
	decoded, err := url.QueryUnescape(rawURL)
	if err != nil {
		return "", err
	}

	// Validate and re-encode
	parsed, err := url.Parse(decoded)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid URL")
	}

	// Only allow HTTP and HTTPS schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("only HTTP and HTTPS URLs are allowed")
	}

	// Re-encode to handle any special characters properly
	return parsed.String(), nil
}

// sanitizeCode performs input sanitization on custom codes
func sanitizeCode(code string) string {
	// Trim whitespace
	code = strings.TrimSpace(code)

	// HTML escape to prevent XSS
	code = html.EscapeString(code)

	// Only allow alphanumeric characters (base62)
	reg := regexp.MustCompile(`[^a-zA-Z0-9]`)
	code = reg.ReplaceAllString(code, "")

	return strings.ToLower(code)
}

// APIHandler handles REST API requests
type APIHandler struct {
	store      *store.MemoryStore
	shortener  *service.Shortener
	baseURL    string
	expiryHrs  int
	mu         sync.Mutex
	rateLimiter *RateLimiter
}

// NewAPIHandler creates a new APIHandler instance
func NewAPIHandler(baseURL string, expiryHrs int) *APIHandler {
	return &APIHandler{
		store:      store.NewMemoryStore(),
		shortener:  service.NewShortener(7),
		baseURL:    baseURL,
		expiryHrs:  expiryHrs,
		rateLimiter: NewRateLimiter(10, time.Second, 20), // 10 req/sec, burst of 20
	}
}

// NewAPIHandlerWithRateLimit creates a new APIHandler with custom rate limiting
func NewAPIHandlerWithRateLimit(baseURL string, expiryHrs int, rate int, window time.Duration, capacity int) *APIHandler {
	return &APIHandler{
		store:      store.NewMemoryStore(),
		shortener:  service.NewShortener(7),
		baseURL:    baseURL,
		expiryHrs:  expiryHrs,
		rateLimiter: NewRateLimiter(rate, window, capacity),
	}
}

// HandleShorten handles POST /api/shorten
func (h *APIHandler) HandleShorten(w http.ResponseWriter, r *http.Request) {
	// Rate limiting
	clientIP := getClientIP(r)
	if !h.rateLimiter.Allow(clientIP) {
		h.writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests. Please try again later.")
		return
	}

	var req model.ShortenRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body")
		return
	}
	defer r.Body.Close()

	// Sanitize URL input
	sanitizedURL, err := sanitizeURL(req.URL)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_URL", "Invalid URL format")
		return
	}

	// Validate URL
	if sanitizedURL == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_URL", "URL is required")
		return
	}

	// Sanitize custom code if provided
	if req.CustomCode != "" {
		req.CustomCode = sanitizeCode(req.CustomCode)
		if req.CustomCode == "" {
			h.writeError(w, http.StatusBadRequest, "INVALID_CODE", "Custom code must be alphanumeric")
			return
		}
	}

	// Check for duplicate URL (only for non-custom codes)
	if req.CustomCode == "" {
		h.mu.Lock()
		existing, err := h.store.GetByOriginalURL(sanitizedURL)
		if err == nil && existing != nil {
			h.mu.Unlock()
			// Return existing short URL
			existing.ShortURL = fmt.Sprintf("%s/r/%s", h.baseURL, existing.Code)
			h.writeJSON(w, http.StatusOK, model.ShortenResponse{
				ShortURL:    existing.ShortURL,
				Code:        existing.Code,
				OriginalURL: existing.OriginalURL,
				CreatedAt:   existing.CreatedAt,
				ExpiresAt:   existing.ExpiresAt,
				IsDuplicate: true,
			})
			return
		}
		h.mu.Unlock()
	}

	// Handle custom code
	var code string
	if req.CustomCode != "" {
		if err := h.shortener.ValidateCode(req.CustomCode); err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_CODE", err.Error())
			return
		}
		code = req.CustomCode
		h.mu.Lock()
		if h.store.Exists(code) {
			h.mu.Unlock()
			h.writeError(w, http.StatusConflict, "CODE_EXISTS", "Custom code already in use")
			return
		}
		h.mu.Unlock()
	} else {
		// Generate unique code
		h.mu.Lock()
		genCode, err := h.store.GenerateUniqueCode(h.shortener)
		h.mu.Unlock()
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "GENERATION_FAILED", "Failed to generate short code")
			return
		}
		code = genCode
	}

	// Calculate expiration - user-specified takes precedence over global
	var expiresAt *time.Time
	expiryHrs := h.expiryHrs
	if req.ExpiryHours != nil && *req.ExpiryHours > 0 {
		expiryHrs = *req.ExpiryHours
	}
	if expiryHrs > 0 {
		exp := time.Now().Add(time.Duration(expiryHrs) * time.Hour)
		expiresAt = &exp
	}

	// Create record
	h.mu.Lock()
	record, err := h.store.Create(code, sanitizedURL, expiresAt)
	h.mu.Unlock()
	if err != nil {
		if err == store.ErrCodeExists {
			h.writeError(w, http.StatusConflict, "CODE_EXISTS", "Short code already exists")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create short URL")
		return
	}

	record.ShortURL = fmt.Sprintf("%s/r/%s", h.baseURL, code)

	response := model.ShortenResponse{
		ShortURL:    record.ShortURL,
		Code:        record.Code,
		OriginalURL: record.OriginalURL,
		CreatedAt:   record.CreatedAt,
		ExpiresAt:   record.ExpiresAt,
	}

	h.writeJSON(w, http.StatusCreated, response)
}

// HandleList handles GET /api/urls
func (h *APIHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	// Rate limiting
	clientIP := getClientIP(r)
	if !h.rateLimiter.Allow(clientIP) {
		h.writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests. Please try again later.")
		return
	}

	records := h.store.List()

	// Fill in ShortURL for each record
	for i := range records {
		records[i].ShortURL = fmt.Sprintf("%s/r/%s", h.baseURL, records[i].Code)
	}

	response := model.URLListResponse{
		URLs:  records,
		Total: len(records),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleStats handles GET /api/stats/:code
func (h *APIHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	// Rate limiting
	clientIP := getClientIP(r)
	if !h.rateLimiter.Allow(clientIP) {
		h.writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests. Please try again later.")
		return
	}

	vars := mux.Vars(r)
	code := sanitizeCode(vars["code"])

	if code == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_CODE", "Invalid code format")
		return
	}

	record, err := h.store.Get(code)
	if err != nil {
		if err == store.ErrNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Short URL not found")
			return
		}
		if err == store.ErrCodeExpired {
			h.writeError(w, http.StatusGone, "EXPIRED", "Short URL has expired")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve stats")
		return
	}

	record.ShortURL = fmt.Sprintf("%s/r/%s", h.baseURL, record.Code)

	h.writeJSON(w, http.StatusOK, record)
}

// HandleDelete handles DELETE /api/urls/:code
func (h *APIHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	// Rate limiting
	clientIP := getClientIP(r)
	if !h.rateLimiter.Allow(clientIP) {
		h.writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests. Please try again later.")
		return
	}

	vars := mux.Vars(r)
	code := sanitizeCode(vars["code"])

	if code == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_CODE", "Invalid code format")
		return
	}

	h.mu.Lock()
	err := h.store.Delete(code)
	h.mu.Unlock()
	if err != nil {
		if err == store.ErrNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Short URL not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete short URL")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleRedirect handles GET /r/:code
func (h *APIHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := sanitizeCode(vars["code"])

	if code == "" {
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}

	record, err := h.store.Get(code)
	if err != nil {
		if err == store.ErrNotFound {
			http.Error(w, "Short URL not found", http.StatusNotFound)
			return
		}
		if err == store.ErrCodeExpired {
			http.Error(w, "Short URL has expired", http.StatusGone)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Increment clicks
	h.store.IncrementClicks(code)

	// Redirect to original URL
	http.Redirect(w, r, record.OriginalURL, http.StatusMovedPermanently)
}

// getClientIP extracts the client IP for rate limiting
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := splitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// splitHostPort splits a host:port string
func splitHostPort(addr string) (string, string, error) {
	// Simple implementation for IPv4
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			// Check if this is an IPv6 address
			if i > 0 && addr[i-1] == ']' {
				continue
			}
			return addr[:i], addr[i+1:], nil
		}
	}
	return addr, "", nil
}

// writeJSON writes a JSON response
func (h *APIHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func (h *APIHandler) writeError(w http.ResponseWriter, status int, err string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.ErrorResponse{
		Error:   err,
		Message: message,
	})
}

// HealthCheck handles GET /health
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}
