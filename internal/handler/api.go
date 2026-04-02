package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/goshort/goshort/internal/model"
	"github.com/goshort/goshort/internal/service"
	"github.com/goshort/goshort/internal/store"
)

// APIHandler handles REST API requests
type APIHandler struct {
	store     *store.MemoryStore
	shortener *service.Shortener
	baseURL   string
	expiryHrs int
	mu        sync.Mutex
}

// NewAPIHandler creates a new APIHandler instance
func NewAPIHandler(baseURL string, expiryHrs int) *APIHandler {
	return &APIHandler{
		store:     store.NewMemoryStore(),
		shortener: service.NewShortener(7),
		baseURL:   baseURL,
		expiryHrs: expiryHrs,
	}
}

// HandleShorten handles POST /api/shorten
func (h *APIHandler) HandleShorten(w http.ResponseWriter, r *http.Request) {
	var req model.ShortenRequest
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body")
		return
	}
	defer r.Body.Close()

	// Validate URL
	if req.URL == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_URL", "URL is required")
		return
	}

	parsedURL, err := url.Parse(req.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_URL", "Invalid URL format")
		return
	}

	// Handle custom code
	var code string
	if req.CustomCode != "" {
		if err := h.shortener.ValidateCode(req.CustomCode); err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_CODE", err.Error())
			return
		}
		code = req.CustomCode
		if h.store.Exists(code) {
			h.writeError(w, http.StatusConflict, "CODE_EXISTS", "Custom code already in use")
			return
		}
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

	// Calculate expiration
	var expiresAt *time.Time
	if h.expiryHrs > 0 {
		exp := time.Now().Add(time.Duration(h.expiryHrs) * time.Hour)
		expiresAt = &exp
	}

	// Create record
	record, err := h.store.Create(code, req.URL, expiresAt)
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
	vars := mux.Vars(r)
	code := vars["code"]

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
	vars := mux.Vars(r)
	code := vars["code"]

	err := h.store.Delete(code)
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
	code := vars["code"]

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
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Increment clicks
	h.store.IncrementClicks(code)

	// Redirect to original URL
	http.Redirect(w, r, record.OriginalURL, http.StatusMovedPermanently)
}

// writeJSON writes a JSON response
func (h *APIHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func (h *APIHandler) writeError(w http.ResponseWriter, status int, error, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.ErrorResponse{
		Error:   error,
		Message: message,
	})
}

// HealthCheck handles GET /health
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}
