package service

import (
	"testing"
)

func TestShortener_Encode(t *testing.T) {
	s := NewShortener(7)

	tests := []struct {
		name string
		id   int64
	}{
		{"zero", 0},
		{"one", 1},
		{"sixty one", 61},
		{"sixty two", 62},
		{"large number", 999999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := s.Encode(tt.id)
			if len(code) != 7 {
				t.Errorf("expected length 7, got %d", len(code))
			}
		})
	}
}

func TestShortener_Decode(t *testing.T) {
	s := NewShortener(7)

	tests := []struct {
		name     string
		code     string
		expected int64
		hasError bool
	}{
		{"zero", "0000000", 0, false},
		{"one", "0000001", 1, false},
		{"base62 max single digit", "000000z", 61, false},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.Decode(tt.code)
			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestShortener_EncodeDecode(t *testing.T) {
	s := NewShortener(7)

	tests := []int64{0, 1, 42, 100, 12345, 999999, 123456789}

	for _, id := range tests {
		code := s.Encode(id)
		decoded, err := s.Decode(code)
		if err != nil {
			t.Errorf("Decode(%s) returned error: %v", code, err)
			continue
		}
		if decoded != id {
			t.Errorf("round trip failed: %d -> %s -> %d", id, code, decoded)
		}
	}
}

func TestShortener_Generate(t *testing.T) {
	s := NewShortener(7)

	code, err := s.Generate()
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	if len(code) != 7 {
		t.Errorf("expected length 7, got %d", len(code))
	}

	// Verify it's base62
	for _, c := range code {
		if !isBase62(c) {
			t.Errorf("Generate() produced non-base62 character: %c", c)
		}
	}
}

func TestShortener_Generate_Uniqueness(t *testing.T) {
	s := NewShortener(7)

	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := s.Generate()
		if err != nil {
			t.Fatalf("Generate() returned error: %v", err)
		}
		if codes[code] {
			t.Errorf("Generate() produced duplicate code: %s", code)
		}
		codes[code] = true
	}
}

func TestShortener_ValidateCode(t *testing.T) {
	s := NewShortener(7)

	tests := []struct {
		name     string
		code     string
		hasError bool
	}{
		{"empty is valid", "", false},
		{"short valid code", "abc", false},
		{"full length valid code", "Abc123", false},
		{"too long", "abcdefgh", true},
		{"with special chars", "abc-123", true},
		{"with spaces", "abc 123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.ValidateCode(tt.code)
			if tt.hasError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestShortener_GetLength(t *testing.T) {
	tests := []int{0, -1, 5, 7, 10}

	for _, length := range tests {
		s := NewShortener(length)
		expected := length
		if expected <= 0 {
			expected = 7
		}
		if s.GetLength() != expected {
			t.Errorf("NewShortener(%d).GetLength() = %d, want %d", length, s.GetLength(), expected)
		}
	}
}

func isBase62(c rune) bool {
	return (c >= '0' && c <= '9') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z')
}
