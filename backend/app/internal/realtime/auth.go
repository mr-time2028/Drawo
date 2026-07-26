package realtime

import (
	"context"
	"fmt"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
	"drawo/pkg/security"
)

// Authenticator centralizes WebSocket token validation.
//
// WebSocket connections are long-lived, so we validate a short-lived access
// token during the initial handshake and again during re-auth. The socket is
// also tied to the server-side session, so logout/ban/reuse detection can revoke
// the connection even before the access token naturally expires.
type Authenticator struct {
	jwt      *security.JWTManager
	sessions repositories.SessionRepository
}

func NewAuthenticator(cfg config.Config, sessions repositories.SessionRepository) *Authenticator {
	return &Authenticator{
		jwt: security.NewJWTManager(
			cfg.App.SecretKey,
			cfg.Auth.Issuer,
			cfg.Auth.AccessTokenExpiry,
			cfg.Auth.RefreshTokenExpiry,
		),
		sessions: sessions,
	}
}

func (a *Authenticator) AuthenticateAccessToken(ctx context.Context, token string) (*AuthContext, error) {
	claims, err := a.jwt.ParseAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	session, err := a.sessions.Get(ctx, claims.SessionID)
	if err != nil || session == nil {
		return nil, fmt.Errorf("session no longer active")
	}
	if session.UserID != claims.UserID {
		return nil, fmt.Errorf("token/session user mismatch")
	}
	if session.IsExpired() {
		return nil, fmt.Errorf("session expired")
	}
	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("access token expiry missing")
	}

	return &AuthContext{
		UserID:          claims.UserID,
		SessionID:       claims.SessionID,
		TokenID:         claims.TokenID,
		AccessExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (a *Authenticator) SessionActive(ctx context.Context, auth *AuthContext) bool {
	if auth == nil {
		return false
	}
	userID, sessionID, _, _ := auth.Snapshot()
	session, err := a.sessions.Get(ctx, sessionID)
	return err == nil && session != nil && session.UserID == userID && !session.IsExpired()
}
