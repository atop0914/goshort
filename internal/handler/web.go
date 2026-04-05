package handler

import (
	"net/http"
	"path/filepath"
)

// WebHandler handles web UI requests
type WebHandler struct {
	baseDir string
}

// NewWebHandler creates a new web handler
func NewWebHandler(baseDir string) *WebHandler {
	return &WebHandler{
		baseDir: baseDir,
	}
}

// Index handles GET /
func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	tmplPath := filepath.Join(h.baseDir, "templates", "index.html")
	http.ServeFile(w, r, tmplPath)
}

// Stats handles GET /stats
func (h *WebHandler) Stats(w http.ResponseWriter, r *http.Request) {
	tmplPath := filepath.Join(h.baseDir, "templates", "stats.html")
	http.ServeFile(w, r, tmplPath)
}
