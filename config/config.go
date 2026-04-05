package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Host        string `json:"host" yaml:"host"`
	Port        int    `json:"port" yaml:"port"`
	BaseURL     string `json:"base_url" yaml:"base_url"`
	ExpiryHours int    `json:"expiry_hours" yaml:"expiry_hours"`
}

// Load reads configuration from a JSON or YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	var cfg Config

	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	} else {
		// Default to JSON
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
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
