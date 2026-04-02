package config

import (
	"encoding/json"
	"os"
)

// Config represents the application configuration
type Config struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	BaseURL     string `json:"base_url"`
	ExpiryHours int    `json:"expiry_hours"`
}

// Load reads configuration from a JSON file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Host:        "0.0.0.0",
		Port:        8080,
		BaseURL:     "http://localhost:8080",
		ExpiryHours: 720, // 30 days
	}
}
