package config

import (
	"encoding/json"
	"os"
	"time"
)

// Config holds application configuration
type Config struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	BaseURL     string `json:"base_url"`
	ExpiryHours int    `json:"expiry_hours"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Host:        "0.0.0.0",
		Port:        8080,
		BaseURL:     "http://localhost:8080",
		ExpiryHours: 720, // 30 days
	}
}

// Load reads configuration from a JSON file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetExpiryDuration returns the expiry duration
func (c *Config) GetExpiryDuration() time.Duration {
	if c.ExpiryHours <= 0 {
		return 0
	}
	return time.Duration(c.ExpiryHours) * time.Hour
}

// GetAddress returns the host:port address
func (c *Config) GetAddress() string {
	return c.Host + ":" + string(rune(c.Port))
}
