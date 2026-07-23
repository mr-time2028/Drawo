package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWSErrorCode(t *testing.T) {
	assert.Equal(t, "auth_failed", WSErrAuthFailed.String())
	assert.Equal(t, "authentication failed", WSDefaultMessage(WSErrAuthFailed))
	assert.Equal(t, "message contains prohibited words", WSDefaultMessage(WSErrBadWord))
	assert.Equal(t, "websocket error", WSDefaultMessage(WSErrorCode("unknown")))
}
