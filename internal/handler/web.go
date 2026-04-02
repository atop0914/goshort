package handler

import (
	"net/http"

	"github.com/goshort/goshort/internal/store"
)

// WebHandler handles web UI requests
type WebHandler struct {
	store   *store.MemoryStore
	baseURL string
}

// NewWebHandler creates a new web handler
func NewWebHandler(s *store.MemoryStore, baseURL string) *WebHandler {
	return &WebHandler{
		store:   s,
		baseURL: baseURL,
	}
}

// Index handles GET /
func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

// Stats handles GET /stats
func (h *WebHandler) Stats(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/stats.html")
}
