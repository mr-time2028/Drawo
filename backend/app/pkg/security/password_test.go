package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerifyPassword(t *testing.T) {
	password := "superSecret123!"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Correct password should verify.
	assert.NoError(t, VerifyPassword(hash, password))

	// Incorrect password should fail.
	assert.Error(t, VerifyPassword(hash, "wrong-password"))
}

func TestGenerateRandomToken(t *testing.T) {
	token, err := GenerateRandomToken(32)
	require.NoError(t, err)
	assert.Len(t, token, 64)

	// Token of size 0
	token, err = GenerateRandomToken(0)
	require.NoError(t, err)
	assert.Len(t, token, 0)
}
