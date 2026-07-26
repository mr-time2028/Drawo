package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOTP_IsExpired(t *testing.T) {
	o := &OTP{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	assert.True(t, o.IsExpired())

	o.ExpiresAt = time.Now().Add(1 * time.Hour)
	assert.False(t, o.IsExpired())
}
