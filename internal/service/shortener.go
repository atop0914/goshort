package service

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
)

const (
	// Base62 charset: 0-9, A-Z, a-z
	charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	// Default short code length
	defaultLength = 7
)

var (
	ErrInvalidLength = errors.New("invalid length")
	ErrEmptyCode     = errors.New("empty code")
)

// Shortener handles short code generation
type Shortener struct {
	length int
}

// NewShortener creates a new Shortener instance
func NewShortener(length int) *Shortener {
	if length <= 0 {
		length = defaultLength
	}
	return &Shortener{length: length}
}

// Encode converts a numeric ID to a base62 string
func (s *Shortener) Encode(id int64) string {
	if id < 0 {
		return ""
	}

	// Handle zero case
	if id == 0 {
		return strings.Repeat(string(charset[0]), s.length)
	}

	var digits []byte
	base := int64(len(charset))

	for id > 0 {
		idx := id % base
		digits = append(digits, charset[idx])
		id /= base
	}

	// Reverse to get most significant digit first
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	// Pad with leading zeros if needed
	if len(digits) < s.length {
		padding := make([]byte, s.length-len(digits))
		for i := range padding {
			padding[i] = charset[0]
		}
		digits = append(padding, digits...)
	}

	return string(digits)
}

// Decode converts a base62 string back to numeric ID
func (s *Shortener) Decode(code string) (int64, error) {
	if code == "" {
		return 0, ErrEmptyCode
	}
	
	code = strings.TrimPrefix(code, " ")
	var result int64
	base := int64(len(charset))
	
	for _, c := range code {
		idx := strings.IndexRune(charset, c)
		if idx == -1 {
			return 0, errors.New("invalid character in code")
		}
		result = result*base + int64(idx)
	}
	
	return result, nil
}

// Generate creates a random short code
func (s *Shortener) Generate() (string, error) {
	result := make([]byte, s.length)
	charsetLen := big.NewInt(int64(len(charset)))
	
	for i := 0; i < s.length; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}
	
	return string(result), nil
}

// ValidateCode checks if a custom code is valid (alphanumeric only)
func (s *Shortener) ValidateCode(code string) error {
	if code == "" {
		return nil // Empty is allowed (will use generated)
	}
	
	if len(code) > s.length {
		return ErrInvalidLength
	}
	
	for _, c := range code {
		if !strings.ContainsRune(charset, c) {
			return errors.New("code must be base62 alphanumeric")
		}
	}
	
	return nil
}

// GetLength returns the configured length
func (s *Shortener) GetLength() int {
	return s.length
}
