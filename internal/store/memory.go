package store

import (
	"errors"
	"sync"
	"time"

	"github.com/goshort/goshort/internal/model"
)

var (
	ErrNotFound      = errors.New("URL not found")
	ErrCodeExists    = errors.New("code already exists")
	ErrCodeExpired   = errors.New("URL has expired")
	ErrInvalidURL    = errors.New("invalid URL")
	ErrEmptyURL      = errors.New("URL cannot be empty")
)

// MemoryStore is a thread-safe in-memory store for URL records
type MemoryStore struct {
	mu    sync.RWMutex
	urls  map[string]*model.URLRecord
	index int64 // Counter for generating sequential IDs
}

// NewMemoryStore creates a new MemoryStore instance
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		urls:  make(map[string]*model.URLRecord),
		index: 0,
	}
}

// nextID generates the next sequential ID
func (s *MemoryStore) nextID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index++
	return s.index
}

// Create adds a new URL record to the store
func (s *MemoryStore) Create(code, originalURL string, expiresAt *time.Time) (*model.URLRecord, error) {
	if originalURL == "" {
		return nil, ErrEmptyURL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if code already exists
	if _, exists := s.urls[code]; exists {
		return nil, ErrCodeExists
	}

	record := &model.URLRecord{
		Code:        code,
		OriginalURL: originalURL,
		ShortURL:    "", // Will be set by handler
		Clicks:      0,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   expiresAt,
	}

	s.urls[code] = record
	return record, nil
}

// Get retrieves a URL record by code
func (s *MemoryStore) Get(code string) (*model.URLRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.urls[code]
	if !exists {
		return nil, ErrNotFound
	}

	// Check expiration
	if record.ExpiresAt != nil && time.Now().After(*record.ExpiresAt) {
		return nil, ErrCodeExpired
	}

	return record, nil
}

// IncrementClicks increments the click counter for a URL
func (s *MemoryStore) IncrementClicks(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.urls[code]
	if !exists {
		return ErrNotFound
	}

	record.Clicks++
	return nil
}

// List returns all URL records
func (s *MemoryStore) List() []model.URLRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]model.URLRecord, 0, len(s.urls))
	for _, record := range s.urls {
		records = append(records, *record)
	}
	return records
}

// Delete removes a URL record from the store
func (s *MemoryStore) Delete(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.urls[code]; !exists {
		return ErrNotFound
	}

	delete(s.urls, code)
	return nil
}

// Exists checks if a code already exists
func (s *MemoryStore) Exists(code string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.urls[code]
	return exists
}

// GetByOriginalURL finds a URL record by original URL
func (s *MemoryStore) GetByOriginalURL(originalURL string) (*model.URLRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, record := range s.urls {
		if record.OriginalURL == originalURL {
			// Check expiration
			if record.ExpiresAt != nil && time.Now().After(*record.ExpiresAt) {
				continue
			}
			return record, nil
		}
	}
	return nil, ErrNotFound
}

// GenerateUniqueCode generates a unique code that doesn't exist in the store
func (s *MemoryStore) GenerateUniqueCode(shortener interface{ Generate() (string, error) }) (string, error) {
	for attempts := 0; attempts < 100; attempts++ {
		code, err := shortener.Generate()
		if err != nil {
			return "", err
		}
		if !s.Exists(code) {
			return code, nil
		}
	}
	return "", errors.New("failed to generate unique code after 100 attempts")
}
