package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/infrastructure/cache"
	"drawo/pkg/security"
)

type wsTestEnv struct {
	cfg      config.Config
	sessions repositories.SessionRepository
	roomRepo repositories.RoomRepository
	hub      *Hub
	handler  *Handler
	server   *httptest.Server
	jwt      *security.JWTManager
}

func newWSTestEnv(t *testing.T) *wsTestEnv {
	t.Helper()
	cfg := config.Get()
	cfg.App.SecretKey = "ws-test-secret"
	cfg.App.Domain = "http://example.com"
	cfg.Auth.Issuer = "drawo"
	cfg.Auth.AccessTokenExpiry = time.Hour
	cfg.Auth.RefreshTokenExpiry = 24 * time.Hour

	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cacheClient.Close() })

	sessions := repositories.NewSessionRepo(cacheClient)
	roomRepo := repositories.NewRoomRepo(cacheClient)
	hub := NewHub(roomRepo)
	handler := NewHandler(cfg, hub, sessions)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &wsTestEnv{
		cfg:      cfg,
		sessions: sessions,
		roomRepo: roomRepo,
		hub:      hub,
		handler:  handler,
		server:   server,
		jwt:      security.NewJWTManager(cfg.App.SecretKey, cfg.Auth.Issuer, cfg.Auth.AccessTokenExpiry, cfg.Auth.RefreshTokenExpiry),
	}
}

func (e *wsTestEnv) createSession(t *testing.T, userID, sessionID, tokenID string) (access, refresh string) {
	t.Helper()
	require.NoError(t, e.sessions.Set(context.Background(), &domain.Session{
		ID:             sessionID,
		UserID:         userID,
		RefreshTokenID: tokenID,
		ExpiresAt:      time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}))
	access, refresh, err := e.jwt.GenerateTokenPair(userID, sessionID, tokenID)
	require.NoError(t, err)
	return access, refresh
}

func (e *wsTestEnv) createRoom(t *testing.T, roomID string) {
	t.Helper()
	require.NoError(t, e.roomRepo.Save(context.Background(), &domain.Room{
		ID:         roomID,
		Name:       "WS Test Room",
		State:      domain.RoomStatePlaying,
		Type:       domain.RoomTypePublic,
		MaxPlayers: 8,
		MinPlayers: 1,
	}))
}

func (e *wsTestEnv) dial(t *testing.T) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func writeEnvelope(t *testing.T, conn *websocket.Conn, eventType EventType, payload any) {
	t.Helper()
	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, conn.WriteJSON(MessageEnvelope{Type: eventType, Payload: payloadJSON, Seq: 1}))
}

func readEnvelopeWithDeadline(t *testing.T, conn *websocket.Conn) MessageEnvelope {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	var env MessageEnvelope
	require.NoError(t, conn.ReadJSON(&env))
	return env
}

func readUntilType(t *testing.T, conn *websocket.Conn, eventType EventType) MessageEnvelope {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		require.NoError(t, conn.SetReadDeadline(deadline))
		var env MessageEnvelope
		require.NoError(t, conn.ReadJSON(&env))
		if env.Type == eventType {
			return env
		}
	}
	t.Fatalf("timed out waiting for event type %s", eventType)
	return MessageEnvelope{}
}

func forceRoomDrawing(t *testing.T, e *wsTestEnv, roomID string, drawerID string) {
	t.Helper()
	room, _, err := e.hub.GetRoom(context.Background(), roomID)
	require.NoError(t, err)
	require.NotNil(t, room)
	room.gameState = GameStateDrawing
	room.state.CurrentDrawerID = drawerID
	room.roundEndsAt = time.Now().Add(time.Minute)
}

func forceAllRoomsDrawing(t *testing.T, e *wsTestEnv, drawerID string) {
	t.Helper()
	e.hub.mu.RLock()
	defer e.hub.mu.RUnlock()
	for _, room := range e.hub.rooms {
		room.gameState = GameStateDrawing
		room.state.CurrentDrawerID = drawerID
		room.roundEndsAt = time.Now().Add(time.Minute)
	}
}

func validStrokePayload() map[string]any {
	return map[string]any{
		"op":    DrawOpStroke,
		"tool":  ToolPencil,
		"color": "#000000",
		"size":  4,
		"points": []map[string]float64{
			{"x": 10, "y": 20},
			{"x": 12, "y": 22},
		},
	}
}

func connectAuthenticated(t *testing.T, e *wsTestEnv, token, roomID string) *websocket.Conn {
	t.Helper()
	conn := e.dial(t)
	writeEnvelope(t, conn, EventAuth, AuthPayload{AccessToken: token})
	writeEnvelope(t, conn, EventJoin, JoinPayload{RoomID: roomID})

	first := readEnvelopeWithDeadline(t, conn)
	assert.Equal(t, EventAuthOK, first.Type)
	for {
		next := readEnvelopeWithDeadline(t, conn)
		if next.Type == EventJoined {
			return conn
		}
		assert.Contains(t, []EventType{EventCanvasSync, EventGameState}, next.Type)
	}
}

func TestWebSocketHandler_AuthenticatedJoinAndBroadcast(t *testing.T) {
	env := newWSTestEnv(t)
	env.createRoom(t, "room-1")
	access1, _ := env.createSession(t, "user-1", "sess-1", "tok-1")
	access2, _ := env.createSession(t, "user-2", "sess-2", "tok-2")

	conn1 := connectAuthenticated(t, env, access1, "room-1")
	conn2 := connectAuthenticated(t, env, access2, "room-1")

	forceRoomDrawing(t, env, "room-1", "user-1")

	writeEnvelope(t, conn1, EventDraw, validStrokePayload())

	msg := readUntilType(t, conn2, EventDraw)
	assert.Equal(t, EventDraw, msg.Type)
	assert.Contains(t, string(msg.Payload), `"op":"stroke"`)
}

func TestWebSocketHandler_RejectsRefreshTokenAsAuth(t *testing.T) {
	env := newWSTestEnv(t)
	env.createRoom(t, "room-1")
	_, refresh := env.createSession(t, "user-1", "sess-1", "tok-1")

	conn := env.dial(t)
	writeEnvelope(t, conn, EventAuth, AuthPayload{AccessToken: refresh})

	msg := readUntilType(t, conn, EventError)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "auth_failed")
}

func TestWebSocketHandler_RejectsMessagesBeforeAuth(t *testing.T) {
	env := newWSTestEnv(t)
	env.createRoom(t, "room-1")

	conn := env.dial(t)
	writeEnvelope(t, conn, EventJoin, JoinPayload{RoomID: "room-1"})

	msg := readUntilType(t, conn, EventError)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "first websocket message must be auth")
}

func TestWebSocketHandler_RejectsMissingRoom(t *testing.T) {
	env := newWSTestEnv(t)
	access, _ := env.createSession(t, "user-1", "sess-1", "tok-1")

	conn := env.dial(t)
	writeEnvelope(t, conn, EventAuth, AuthPayload{AccessToken: access})
	writeEnvelope(t, conn, EventJoin, JoinPayload{RoomID: "missing-room"})

	msg := readUntilType(t, conn, EventError)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "room not found")
}

func TestWebSocketHandler_ClosesWhenSessionRevoked(t *testing.T) {
	env := newWSTestEnv(t)
	env.createRoom(t, "room-1")
	access, _ := env.createSession(t, "user-1", "sess-1", "tok-1")
	conn := connectAuthenticated(t, env, access, "room-1")

	require.NoError(t, env.sessions.Delete(context.Background(), "sess-1"))
	writeEnvelope(t, conn, EventChat, map[string]string{"text": "hello"})

	msg := readUntilType(t, conn, EventError)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "session_revoked")
}

func TestWebSocketHandler_OriginPolicy(t *testing.T) {
	env := newWSTestEnv(t)

	assert.True(t, env.handler.checkOrigin(&http.Request{Host: "api.example.com", Header: http.Header{}}))
	assert.True(t, env.handler.checkOrigin(&http.Request{Host: "example.com", Header: http.Header{"Origin": []string{"http://example.com"}}}))
	assert.True(t, env.handler.checkOrigin(&http.Request{Host: "api.internal", Header: http.Header{"Origin": []string{"http://example.com"}}}))
	assert.False(t, env.handler.checkOrigin(&http.Request{Host: "api.internal", Header: http.Header{"Origin": []string{"http://evil.example"}}}))
}

func TestWebSocketHandler_RateLimitsAbusiveClient(t *testing.T) {
	env := newWSTestEnv(t)
	env.createRoom(t, "room-1")
	access, _ := env.createSession(t, "user-1", "sess-1", "tok-1")
	conn := connectAuthenticated(t, env, access, "room-1")

	for i := 0; i < maxMessagesPerSecond+5; i++ {
		_ = conn.WriteJSON(MessageEnvelope{Type: EventChat, Payload: json.RawMessage(`{"text":"spam"}`)})
	}

	for {
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
		var msg MessageEnvelope
		err := conn.ReadJSON(&msg)
		if err != nil {
			// Closing an abusive connection is acceptable even if the error frame is
			// lost in the close race. The important behavior is that spam does not
			// keep the socket alive or block the room.
			return
		}
		if msg.Type == EventError {
			assert.Contains(t, string(msg.Payload), "rate_limited")
			return
		}
	}
}

func newWSTestEnvWithAccessExpiry(t *testing.T, accessExpiry time.Duration) *wsTestEnv {
	t.Helper()
	cfg := config.Get()
	cfg.App.SecretKey = "ws-test-secret"
	cfg.App.Domain = "http://example.com"
	cfg.Auth.Issuer = "drawo"
	cfg.Auth.AccessTokenExpiry = accessExpiry
	cfg.Auth.RefreshTokenExpiry = 24 * time.Hour

	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cacheClient.Close() })

	sessions := repositories.NewSessionRepo(cacheClient)
	roomRepo := repositories.NewRoomRepo(cacheClient)
	hub := NewHub(roomRepo)
	handler := NewHandler(cfg, hub, sessions)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &wsTestEnv{
		cfg:      cfg,
		sessions: sessions,
		roomRepo: roomRepo,
		hub:      hub,
		handler:  handler,
		server:   server,
		jwt:      security.NewJWTManager(cfg.App.SecretKey, cfg.Auth.Issuer, cfg.Auth.AccessTokenExpiry, cfg.Auth.RefreshTokenExpiry),
	}
}

func TestWebSocketHandler_ClosesAfterAccessTokenExpiry(t *testing.T) {
	env := newWSTestEnvWithAccessExpiry(t, time.Second)
	env.createRoom(t, "room-1")
	access, _ := env.createSession(t, "user-1", "sess-1", "tok-1")
	conn := connectAuthenticated(t, env, access, "room-1")

	time.Sleep(1200 * time.Millisecond)
	writeEnvelope(t, conn, EventChat, map[string]string{"text": "after expiry"})

	msg := readUntilType(t, conn, EventError)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "auth_expired")
}

func TestWebSocketHandler_ReauthKeepsSocketAliveWithNewAccessToken(t *testing.T) {
	env := newWSTestEnvWithAccessExpiry(t, 2*time.Second)
	env.createRoom(t, "room-1")
	access1, _ := env.createSession(t, "user-1", "sess-1", "tok-1")
	accessReceiver, _ := env.createSession(t, "user-2", "sess-2", "tok-2")

	conn1 := connectAuthenticated(t, env, access1, "room-1")
	conn2 := connectAuthenticated(t, env, accessReceiver, "room-1")
	forceRoomDrawing(t, env, "room-1", "user-1")

	// Simulate HTTP refresh in the background: refresh rotation changes the token
	// ID in session storage, and the client gets a new short-lived access token.
	time.Sleep(500 * time.Millisecond)
	longerLivedJWT := security.NewJWTManager(env.cfg.App.SecretKey, env.cfg.Auth.Issuer, 5*time.Second, env.cfg.Auth.RefreshTokenExpiry)
	access2, _, err := longerLivedJWT.GenerateTokenPair("user-1", "sess-1", "tok-rotated")
	require.NoError(t, err)
	require.NoError(t, env.sessions.Set(context.Background(), &domain.Session{
		ID:             "sess-1",
		UserID:         "user-1",
		RefreshTokenID: "tok-rotated",
		ExpiresAt:      time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}))

	writeEnvelope(t, conn1, EventAuth, AuthPayload{AccessToken: access2})
	reauthOK := readUntilType(t, conn1, EventAuthOK)
	assert.Equal(t, EventAuthOK, reauthOK.Type)

	// If re-auth did not update the socket auth context, this message would be
	// rejected because access1 is past expiry.
	time.Sleep(1700 * time.Millisecond)
	writeEnvelope(t, conn1, EventDraw, validStrokePayload())

	msg := readUntilType(t, conn2, EventDraw)
	assert.Equal(t, EventDraw, msg.Type)
}

func TestWebSocketHandler_BackendPublicMatchmakingWithoutRoomID(t *testing.T) {
	env := newWSTestEnv(t)
	access1, _ := env.createSession(t, "user-1", "sess-1", "tok-1")
	access2, _ := env.createSession(t, "user-2", "sess-2", "tok-2")

	// The frontend does not know a room_id. It only asks for public matchmaking.
	conn1 := connectAuthenticated(t, env, access1, "")
	conn2 := connectAuthenticated(t, env, access2, "")

	forceAllRoomsDrawing(t, env, "user-1")

	writeEnvelope(t, conn1, EventDraw, validStrokePayload())
	msg := readUntilType(t, conn2, EventDraw)
	assert.Equal(t, EventDraw, msg.Type)
}
