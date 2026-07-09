// Package repositories defines how data is persisted and retrieved.
// This file implements the "Single Device Policy" using a token-mapping strategy.
package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"drawo/internal/core/domain"
)

// SessionRepository defines the contract for managing active user sessions.
type SessionRepository interface {
	// Set stores a session and enforces the Single Device Policy by revoking old ones.
	Set(ctx context.Context, session *domain.Session) error
	// Get retrieves a session by its unique Session ID.
	Get(ctx context.Context, sessionID string) (*domain.Session, error)
	// Delete removes a single session (Logout).
	Delete(ctx context.Context, sessionID string) error
	// DeleteAllForUser removes any active session for a specific user.
	DeleteAllForUser(ctx context.Context, userID string) error
}

// sessionRepo implements SessionRepository using a CacheRepository (Redis/Memory).
type sessionRepo struct {
	cache CacheRepository
}

// NewSessionRepo creates a new session repository.
func NewSessionRepo(cache CacheRepository) SessionRepository {
	return &sessionRepo{cache: cache}
}

// Key constants for Redis storage.
const (
	// sess:<SESSION_ID> -> Stores the full JSON (UserID, IP, Device Name)
	sessionKeyPrefix = "sess:"
	// active_sess:<USER_ID> -> Stores only the current valid SESSION_ID
	activeSessionPrefix = "active_sess:"
)

// Set implements the Single Device Policy: "Last Login Wins".
func (r *sessionRepo) Set(ctx context.Context, session *domain.Session) error {
	// 1. Logic Check: Ensure we aren't saving an already expired session.
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("cannot save session: already expired")
	}

	// 2. ENFORCE SINGLE DEVICE POLICY:
	// We look at our "Pointer" to see if this user already has an active Session ID.
	pointerKey := activeSessionPrefix + session.UserID
	oldSessionID, err := r.cache.Get(ctx, pointerKey)
	
	if err == nil && oldSessionID != "" {
		// An old session exists! 
		// We delete its details so the old JWT becomes useless.
		_ = r.cache.Delete(ctx, sessionKeyPrefix+oldSessionID)
	}

	// 3. Prepare the new session details (JSON).
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	// 4. Save the new session record.
	key := sessionKeyPrefix + session.ID
	if err := r.cache.Set(ctx, key, string(data), ttl); err != nil {
		return err
	}

	// 5. Update the pointer so we know this is now the "Active" session for this user.
	return r.cache.Set(ctx, pointerKey, session.ID, ttl)
}

// Get fetches the session by its unique ID.
func (r *sessionRepo) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	data, err := r.cache.Get(ctx, sessionKeyPrefix+sessionID)
	if err != nil {
		return nil, nil // Session was either deleted by the Single Device Policy or expired.
	}

	var session domain.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &session, nil
}

// Delete removes a specific session (used for manual Logout).
func (r *sessionRepo) Delete(ctx context.Context, sessionID string) error {
	// 1. Find the session so we know which UserID pointer to clean up.
	sess, _ := r.Get(ctx, sessionID)
	if sess != nil {
		_ = r.cache.Delete(ctx, activeSessionPrefix+sess.UserID)
	}
	// 2. Delete the details.
	return r.cache.Delete(ctx, sessionKeyPrefix+sessionID)
}

// DeleteAllForUser ensures no active session exists for a specific user.
func (r *sessionRepo) DeleteAllForUser(ctx context.Context, userID string) error {
	pointerKey := activeSessionPrefix + userID
	oldSessionID, err := r.cache.Get(ctx, pointerKey)
	if err == nil && oldSessionID != "" {
		_ = r.cache.Delete(ctx, sessionKeyPrefix+oldSessionID)
	}
	return r.cache.Delete(ctx, pointerKey)
}
