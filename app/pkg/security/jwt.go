// Package security contains security primitives used across the application.
package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the data stored inside our JWT tokens.
type Claims struct {
	UserID    string `json:"sub"` // Subject (Standard)
	SessionID string `json:"sid"` // Drawo specific: Link to Redis session
	TokenID   string `json:"tid"` // Drawo specific: For rotation detection
	jwt.RegisteredClaims
}

// JWTManager handles signing and parsing of tokens.
type JWTManager struct {
	secretKey     []byte
	issuer        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewJWTManager creates a new manager with the provided configuration.
func NewJWTManager(secret string, issuer string, accessExp, refreshExp time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secret),
		issuer:        issuer,
		accessExpiry:  accessExp,
		refreshExpiry: refreshExp,
	}
}

// GenerateTokenPair creates both an Access and a Refresh token.
func (m *JWTManager) GenerateTokenPair(userID, sessionID, tokenID string) (string, string, error) {
	// 1. Generate Access Token
	accessClaims := &Claims{
		UserID:    userID,
		SessionID: sessionID,
		TokenID:   tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    m.issuer,
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(m.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	// 2. Generate Refresh Token (same IDs, longer expiry)
	refreshClaims := &Claims{
		UserID:    userID,
		SessionID: sessionID,
		TokenID:   tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    m.issuer,
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(m.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// ParseToken validates the token string and returns the claims.
func (m *JWTManager) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
