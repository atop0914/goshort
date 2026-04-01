package store

import (
	"errors"
	"sync"
	"time"

	"goshort/internal/model"
)

var (
	ErrNotFound     = errors.New("URL not found")
	ErrCodeExists   = errors.New("short code already exists")
	ErrURLExists    = errors.New("URL already shortened")
	ErrCodeExpired  = errors.New("short code has expired")
)

// MemoryStore is an in-memory URL storage
type MemoryStore struct {
	mu    sync.RWMutex
	urls  map[string]*model.URLRecord
	byURL map[string]string // original URL -> code
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		urls:  make(map[string]*model.URLRecord),
		byURL: make(map[string]string),
	}
}

// Create adds a new URL record to the store
func (s *MemoryStore) Create(code, originalURL, shortURL string, expiresAt *time.Time) (*model.URLRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if code already exists
	if _, exists := s.urls[code]; exists {
		return nil, ErrCodeExists
	}

	// Check if URL already has a short code
	if _, exists := s.byURL[originalURL]; exists {
		return nil, ErrURLExists
	}

	record := &model.URLRecord{
		Code:        code,
		OriginalURL: originalURL,
		ShortURL:    shortURL,
		Clicks:      0,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   expiresAt,
	}

	s.urls[code] = record
	s.byURL[originalURL] = code

	return record, nil
}

// GetByCode retrieves a URL record by its short code
func (s *MemoryStore) GetByCode(code string) (*model.URLRecord, error) {
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

// IncrementClicks increments the click counter for a code
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

// GetAll returns all URL records
func (s *MemoryStore) GetAll() []model.URLRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]model.URLRecord, 0, len(s.urls))
	for _, record := range s.urls {
		records = append(records, *record)
	}
	return records
}

// Delete removes a URL record by code
func (s *MemoryStore) Delete(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.urls[code]
	if !exists {
		return ErrNotFound
	}

	delete(s.urls, code)
	delete(s.byURL, record.OriginalURL)

	return nil
}

// Exists checks if a code exists in the store
func (s *MemoryStore) Exists(code string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.urls[code]
	return exists
}
