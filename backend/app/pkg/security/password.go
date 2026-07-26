// Package security contains security primitives used across the application.
//
// Responsibility:
//   - Password hashing and verification.
//   - Cryptographically secure token/secret generation.
//   - Input sanitization helpers.
//
// Why keep this in pkg/security?
//
//	These are cross-cutting concerns. Placing them here prevents every module from
//	importing crypto packages directly and makes them easy to test in isolation.
package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password using bcrypt.
// bcrypt.DefaultCost (10) is a safe balance between CPU cost and login latency.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword compares a bcrypt hash with a plaintext password.
func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateRandomToken returns a cryptographically secure random hex string.
// It is used for refresh tokens, email verification codes, etc.
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
