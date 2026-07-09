package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager(t *testing.T) {
	mgr := NewJWTManager("secret", "drawo", 10*time.Minute, 1*time.Hour)

	// 1. Generate Success
	acc, ref, err := mgr.GenerateTokenPair("user-1", "sess-1", "tok-1")
	require.NoError(t, err)
	assert.NotEmpty(t, acc)
	assert.NotEmpty(t, ref)

	// 2. Parse Valid
	claims, err := mgr.ParseToken(acc)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "sess-1", claims.SessionID)

	// 3. Parse Invalid Format
	_, err = mgr.ParseToken("invalid.token.here")
	assert.Error(t, err)

	// 4. Secret mismatch
	badMgr := NewJWTManager("wrong-secret", "drawo", 10*time.Minute, 1*time.Hour)
	_, err = badMgr.ParseToken(acc)
	assert.Error(t, err)

	// 5. Generate Error (Impossible without mock or invalid key)
	// JWT lib usually doesn't error on HS256 sign unless key is totally wrong type.
}
