package store

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemoryStore_Create(t *testing.T) {
	s := NewMemoryStore()

	record, err := s.Create("test123", "https://example.com", nil)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if record.Code != "test123" {
		t.Errorf("expected code 'test123', got '%s'", record.Code)
	}
	if record.OriginalURL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got '%s'", record.OriginalURL)
	}
}

func TestMemoryStore_Create_DuplicateCode(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.Create("test", "https://example.com", nil)
	if err != nil {
		t.Fatalf("first Create() returned error: %v", err)
	}

	_, err = s.Create("test", "https://example2.com", nil)
	if err != ErrCodeExists {
		t.Errorf("expected ErrCodeExists, got %v", err)
	}
}

func TestMemoryStore_Create_EmptyURL(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.Create("test", "", nil)
	if err != ErrEmptyURL {
		t.Errorf("expected ErrEmptyURL, got %v", err)
	}
}

func TestMemoryStore_Get(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.Create("test123", "https://example.com", nil)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	record, err := s.Get("test123")
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	if record.Code != "test123" {
		t.Errorf("expected code 'test123', got '%s'", record.Code)
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.Get("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_Get_Expired(t *testing.T) {
	s := NewMemoryStore()

	past := time.Now().Add(-1 * time.Hour)
	_, err := s.Create("expired", "https://example.com", &past)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	_, err = s.Get("expired")
	if err != ErrCodeExpired {
		t.Errorf("expected ErrCodeExpired, got %v", err)
	}
}

func TestMemoryStore_IncrementClicks(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.Create("test", "https://example.com", nil)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	for i := 0; i < 5; i++ {
		if err := s.IncrementClicks("test"); err != nil {
			t.Fatalf("IncrementClicks() returned error: %v", err)
		}
	}

	record, _ := s.Get("test")
	if record.Clicks != 5 {
		t.Errorf("expected 5 clicks, got %d", record.Clicks)
	}
}

func TestMemoryStore_List(t *testing.T) {
	s := NewMemoryStore()

	urls := []string{"https://a.com", "https://b.com", "https://c.com"}
	for i, u := range urls {
		code := string(rune('a' + i))
		s.Create(code, u, nil)
	}

	records := s.List()
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()

	s.Create("test", "https://example.com", nil)

	err := s.Delete("test")
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	_, err = s.Get("test")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStore_Delete_NotFound(t *testing.T) {
	s := NewMemoryStore()

	err := s.Delete("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_Exists(t *testing.T) {
	s := NewMemoryStore()

	s.Create("test", "https://example.com", nil)

	if !s.Exists("test") {
		t.Error("expected true for existing code")
	}

	if s.Exists("nonexistent") {
		t.Error("expected false for nonexistent code")
	}
}

func TestMemoryStore_Concurrent(t *testing.T) {
	s := NewMemoryStore()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			code := fmt.Sprintf("code%d", i)
			s.Create(code, "https://example.com", nil)
			s.Get(code)
			s.Exists(code)
			s.IncrementClicks(code)
		}(i)
	}
	wg.Wait()

	if len(s.List()) != 100 {
		t.Errorf("expected 100 records, got %d", len(s.List()))
	}
}
