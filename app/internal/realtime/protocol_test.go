package realtime

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateClientEvent(t *testing.T) {
	validPayload := json.RawMessage(`{"ok":true}`)

	assert.NoError(t, validateClientEvent(&MessageEnvelope{Type: EventChat, Payload: validPayload}))
	assert.NoError(t, validateClientEvent(&MessageEnvelope{Type: EventDraw, Payload: validPayload}))
	assert.NoError(t, validateClientEvent(&MessageEnvelope{Type: EventGame, Payload: validPayload}))
	assert.NoError(t, validateClientEvent(&MessageEnvelope{Type: EventClearCanvas, Payload: validPayload}))
	assert.NoError(t, validateClientEvent(&MessageEnvelope{Type: EventLeave}))

	assert.Error(t, validateClientEvent(&MessageEnvelope{Type: EventChat}))
	assert.Error(t, validateClientEvent(&MessageEnvelope{Type: EventDraw, Payload: json.RawMessage(`not-json`)}))
	assert.Error(t, validateClientEvent(&MessageEnvelope{Type: EventAuth, Payload: validPayload}))
	assert.Error(t, validateClientEvent(&MessageEnvelope{Type: EventJoin, Payload: validPayload}))
	assert.Error(t, validateClientEvent(&MessageEnvelope{Type: EventType("unknown"), Payload: validPayload}))
}

func TestSafeSendAndCloseClientSend(t *testing.T) {
	client := &Client{Send: make(chan []byte, 1), Done: make(chan struct{})}
	assert.True(t, safeSend(client, []byte("one")))
	assert.False(t, safeSend(client, []byte("two")))

	closeClientSend(client)
	select {
	case <-client.Done:
	default:
		t.Fatal("expected Done to be closed")
	}

	// Idempotent: calling again must not panic.
	closeClientSend(client)
}
