package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSession_IsExpired(t *testing.T) {
	s := &Session{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	assert.True(t, s.IsExpired())

	s.ExpiresAt = time.Now().Add(1 * time.Hour)
	assert.False(t, s.IsExpired())
}
