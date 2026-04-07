package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/goshort/goshort/config"
	"github.com/goshort/goshort/internal/handler"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	// Load configuration
	var cfg *config.Config
	if _, err := os.Stat(*configPath); err == nil {
		cfg, err = config.Load(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		log.Printf("Loaded config from %s", *configPath)
	} else {
		cfg = config.Default()
		log.Println("Using default configuration")
	}

	// Get absolute paths for templates and static files
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	baseDir := filepath.Dir(exePath)

	// Create API handler with rate limiting
	rateLimitRate := cfg.RateLimitRate
	if rateLimitRate <= 0 {
		rateLimitRate = 10
	}
	rateLimitCap := cfg.RateLimitCap
	if rateLimitCap <= 0 {
		rateLimitCap = 20
	}
	apiHandler := handler.NewAPIHandlerWithRateLimit(
		cfg.BaseURL,
		cfg.ExpiryHours,
		rateLimitRate,
		time.Second,
		rateLimitCap,
	)

	// Create web handler
	webHandler := handler.NewWebHandler(baseDir)

	// Setup router
	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/api/shorten", apiHandler.HandleShorten).Methods("POST")
	router.HandleFunc("/api/urls", apiHandler.HandleList).Methods("GET")
	router.HandleFunc("/api/urls/{code}", apiHandler.HandleDelete).Methods("DELETE")
	router.HandleFunc("/api/stats/{code}", apiHandler.HandleStats).Methods("GET")

	// Redirect route
	router.HandleFunc("/r/{code}", apiHandler.HandleRedirect).Methods("GET")

	// Health check
	router.HandleFunc("/health", apiHandler.HealthCheck).Methods("GET")

	// Web UI routes
	router.HandleFunc("/", webHandler.Index).Methods("GET")
	router.HandleFunc("/stats", webHandler.Stats).Methods("GET")

	// Static files
	staticDir := filepath.Join(baseDir, "static")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("Starting GoShort server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
