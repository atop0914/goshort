package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

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

	// Create API handler
	apiHandler := handler.NewAPIHandler(cfg.BaseURL, cfg.ExpiryHours)

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

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("Starting GoShort server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
