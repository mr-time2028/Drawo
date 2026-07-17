// Package security contains security primitives used across the application.
package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// TokenTypeAccess is accepted by HTTP middleware and WebSocket handshakes.
	TokenTypeAccess = "access"
	// TokenTypeRefresh is accepted only by the refresh endpoint.
	TokenTypeRefresh = "refresh"
)

// Claims represents the data stored inside our JWT tokens.
type Claims struct {
	UserID    string `json:"sub"` // Subject (standard): authenticated user ID.
	SessionID string `json:"sid"` // Drawo-specific: active Redis/cache session ID.
	TokenID   string `json:"tid"` // Drawo-specific: refresh rotation identifier.
	TokenType string `json:"typ"` // "access" or "refresh"; prevents token confusion attacks.
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
	now := time.Now()

	accessClaims := &Claims{
		UserID:    userID,
		SessionID: sessionID,
		TokenID:   tokenID,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    m.issuer,
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(m.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := &Claims{
		UserID:    userID,
		SessionID: sessionID,
		TokenID:   tokenID,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    m.issuer,
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(m.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// ParseToken validates token signature, expiry, issuer, and returns the claims.
//
// SECURITY NOTE:
// Prefer ParseAccessToken or ParseRefreshToken at call sites. ParseToken is kept
// for generic tooling/tests, but application boundaries should enforce token type
// to avoid accepting a long-lived refresh token where a short-lived access token
// is required.
func (m *JWTManager) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secretKey, nil
	}, jwt.WithIssuer(m.issuer))

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.UserID == "" || claims.SessionID == "" || claims.TokenID == "" || claims.TokenType == "" {
		return nil, fmt.Errorf("missing required token claims")
	}

	return claims, nil
}

// ParseAccessToken accepts only short-lived access tokens.
func (m *JWTManager) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims, err := m.ParseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, fmt.Errorf("unexpected token type: %s", claims.TokenType)
	}
	return claims, nil
}

// ParseRefreshToken accepts only long-lived refresh tokens.
func (m *JWTManager) ParseRefreshToken(tokenStr string) (*Claims, error) {
	claims, err := m.ParseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, fmt.Errorf("unexpected token type: %s", claims.TokenType)
	}
	return claims, nil
}
