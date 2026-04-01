package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"goshort/config"
	"goshort/internal/handler"
	"goshort/internal/store"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Warning: Failed to load config: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}

	// Initialize shared store
	urlStore := store.NewMemoryStore()

	// Initialize handlers
	apiHandler := handler.NewAPIHandler(urlStore, cfg.BaseURL, cfg.ExpiryHours)
	webHandler := handler.NewWebHandler(urlStore, cfg.BaseURL)

	// Setup router
	r := mux.NewRouter()

	// API routes
	r.HandleFunc("/api/shorten", apiHandler.Shorten).Methods(http.MethodPost)
	r.HandleFunc("/api/urls", apiHandler.List).Methods(http.MethodGet)
	r.HandleFunc("/api/stats/{code}", apiHandler.Stats).Methods(http.MethodGet)
	r.HandleFunc("/api/urls/{code}", apiHandler.Delete).Methods(http.MethodDelete)

	// Redirect route
	r.HandleFunc("/r/{code}", apiHandler.Redirect).Methods(http.MethodGet)

	// Web UI routes
	r.HandleFunc("/", webHandler.Index).Methods(http.MethodGet)
	r.HandleFunc("/stats", webHandler.Stats).Methods(http.MethodGet)

	// Static files
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	r.PathPrefix("/static/").Handler(staticHandler)

	addr := cfg.Host + ":" + string(rune(cfg.Port))
	log.Printf("GoShort starting on %s", addr)
	log.Printf("Base URL: %s", cfg.BaseURL)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
