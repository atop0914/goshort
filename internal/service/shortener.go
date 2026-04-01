package service

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
)

const (
	// Base62 characters (alphanumeric)
	charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	base    = int64(len(charset))
)

// Shortener generates short codes for URLs
type Shortener struct{}

// NewShortener creates a new Shortener instance
func NewShortener() *Shortener {
	return &Shortener{}
}

// Generate creates a random short code of given length
func (s *Shortener) Generate(length int) (string, error) {
	if length < 1 || length > 32 {
		return "", errors.New("length must be between 1 and 32")
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(base))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

// ValidateCode checks if a custom code is valid (alphanumeric only, reasonable length)
func (s *Shortener) ValidateCode(code string) error {
	if len(code) < 1 || len(code) > 32 {
		return errors.New("code length must be between 1 and 32")
	}
	for _, c := range code {
		if !strings.ContainsRune(charset, c) {
			return errors.New("code must be alphanumeric (0-9, A-Z, a-z)")
		}
	}
	return nil
}

// EncodeID converts a numeric ID to a base62 string
func (s *Shortener) EncodeID(id int64) string {
	if id == 0 {
		return string(charset[0])
	}

	result := make([]byte, 0)
	for id > 0 {
		idx := id % base
		result = append([]byte{charset[idx]}, result...)
		id /= base
	}
	return string(result)
}
