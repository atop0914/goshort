package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"goshort/internal/model"
	"goshort/internal/service"
	"goshort/internal/store"
)

// APIHandler handles REST API requests
type APIHandler struct {
	store       *store.MemoryStore
	shortener   *service.Shortener
	baseURL     string
	expiryHours int
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(store *store.MemoryStore, baseURL string, expiryHours int) *APIHandler {
	return &APIHandler{
		store:       store,
		shortener:   service.NewShortener(),
		baseURL:     baseURL,
		expiryHours: expiryHours,
	}
}

// Shorten handles POST /api/shorten
func (h *APIHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req model.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	defer r.Body.Close()

	if req.URL == "" {
		h.writeError(w, http.StatusBadRequest, "missing_url", "URL is required")
		return
	}

	var code string
	var err error

	if req.CustomCode != "" {
		if err := h.shortener.ValidateCode(req.CustomCode); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_code", err.Error())
			return
		}
		if h.store.Exists(req.CustomCode) {
			h.writeError(w, http.StatusConflict, "code_exists", "Short code already exists")
			return
		}
		code = req.CustomCode
	} else {
		code, err = h.shortener.Generate(6)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "generation_failed", "Failed to generate short code")
			return
		}
	}

	var expiresAt *time.Time
	if h.expiryHours > 0 {
		t := time.Now().Add(time.Duration(h.expiryHours) * time.Hour)
		expiresAt = &t
	}

	shortURL := fmt.Sprintf("%s/r/%s", h.baseURL, code)

	record, err := h.store.Create(code, req.URL, shortURL, expiresAt)
	if err != nil {
		if err == store.ErrCodeExists {
			h.writeError(w, http.StatusConflict, "code_exists", "Short code already exists")
			return
		}
		if err == store.ErrURLExists {
			h.writeError(w, http.StatusConflict, "url_exists", "URL already has a short link")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "storage_error", err.Error())
		return
	}

	resp := model.ShortenResponse{
		ShortURL:    record.ShortURL,
		Code:        record.Code,
		OriginalURL: record.OriginalURL,
		CreatedAt:   record.CreatedAt,
		ExpiresAt:   record.ExpiresAt,
	}

	h.writeJSON(w, http.StatusCreated, resp)
}

// List handles GET /api/urls
func (h *APIHandler) List(w http.ResponseWriter, r *http.Request) {
	records := h.store.GetAll()
	resp := model.ListResponse{
		URLs:  records,
		Total: len(records),
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// Stats handles GET /api/stats/:code
func (h *APIHandler) Stats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		h.writeError(w, http.StatusBadRequest, "missing_code", "Short code is required")
		return
	}

	record, err := h.store.GetByCode(code)
	if err != nil {
		if err == store.ErrNotFound {
			h.writeError(w, http.StatusNotFound, "not_found", "Short URL not found")
			return
		}
		if err == store.ErrCodeExpired {
			h.writeError(w, http.StatusGone, "expired", "Short URL has expired")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, record)
}

// Delete handles DELETE /api/urls/:code
func (h *APIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		h.writeError(w, http.StatusBadRequest, "missing_code", "Short code is required")
		return
	}

	err := h.store.Delete(code)
	if err != nil {
		if err == store.ErrNotFound {
			h.writeError(w, http.StatusNotFound, "not_found", "Short URL not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Short URL deleted"})
}

// Redirect handles GET /r/:code
func (h *APIHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		http.Error(w, "Short code is required", http.StatusBadRequest)
		return
	}

	record, err := h.store.GetByCode(code)
	if err != nil {
		if err == store.ErrNotFound {
			http.Error(w, "Short URL not found", http.StatusNotFound)
			return
		}
		if err == store.ErrCodeExpired {
			http.Error(w, "Short URL has expired", http.StatusGone)
			return
		}
		log.Printf("Error getting URL: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_ = h.store.IncrementClicks(code)
	log.Printf("Redirect %s -> %s", code, record.OriginalURL)
	http.Redirect(w, r, record.OriginalURL, http.StatusMovedPermanently)
}

func (h *APIHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *APIHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, model.ErrorResponse{
		Error:   code,
		Message: message,
	})
}
