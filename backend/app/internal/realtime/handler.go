package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/pkg/errors"
)

const (
	authDeadline              = 15 * time.Second
	joinDeadline              = 20 * time.Second
	sessionCheckPeriod        = 30 * time.Second
	clientSendBuffer          = 256
	maxMessagesPerSecond      = 40
	maxConsecutiveBadMessages = 3
)

var (
	// The server warns the client before the access-token-bound socket auth expires.
	// Tests may temporarily shorten these package variables.
	reauthWarningBefore = 2 * time.Minute
)

// RoomLookup is the narrow subset of services.RoomService the realtime
// handler needs (to avoid an import cycle: services → realtime is already
// true for hub coordination, so we depend on an interface here).
type RoomLookup interface {
	ValidateGuestToken(ctx context.Context, token string) (*domain.GuestAuth, error)
}

// UserLookup is the narrow repository slice the realtime handler needs to
// resolve a username from a user ID at WebSocket auth time. Accepting an
// interface here avoids coupling the handler to the full UserRepository.
type UserLookup interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

type Handler struct {
	cfg           config.Config
	hub           *Hub
	authenticator *Authenticator
	rooms         RoomLookup
	users         UserLookup
	upgrader      websocket.Upgrader
}

func NewHandler(cfg config.Config, hub *Hub, sessions repositories.SessionRepository, rooms RoomLookup, users UserLookup) *Handler {
	h := &Handler{
		cfg:           cfg,
		hub:           hub,
		authenticator: NewAuthenticator(cfg, sessions),
		rooms:         rooms,
		users:         users,
	}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	conn.SetReadLimit(maxMessageSize)

	authCtx, isGuest, err := h.readAuth(r.Context(), conn)
	if err != nil {
		// writePump has not started yet — write the error/close frame
		// directly on the hijacked conn and return.
		writeCloseError(conn, errors.WSErrAuthFailed, err.Error(), websocket.ClosePolicyViolation)
		return
	}

	userID, sessionID, _, accessExpiresAt := authCtx.Snapshot()
	client := &Client{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		RoomID:    authCtx.RoomID, // pre-binding for guests; hub.JoinRoom will overwrite/confirm.
		Username:  authCtx.Nickname, // for guests this is already set; registered users get their name from hub.
		Conn:      conn,
		Send:      make(chan []byte, clientSendBuffer),
		Done:      make(chan struct{}),
	}

	// Start the write pump BEFORE enqueuing auth_ok or reading `join`.
	// Otherwise a well-behaved client that waits for auth_ok before sending
	// join will deadlock with us (server waits for join, client waits for
	// auth_ok) and surface as "read tcp ... i/o timeout" on the client.
	writeCtx, writeCancel := context.WithCancel(r.Context())
	doneWriting := make(chan struct{})
	go func() {
		h.writePump(writeCtx, client, authCtx)
		close(doneWriting)
	}()

	// shutdownWritePump is the ONE way to tear the writer goroutine down
	// once it is running. It closes client.Send (via closeClientSend which
	// is idempotent and recover()-safe) so writePump drains any queued
	// messages — including an EventError we just enqueued — then sends a
	// proper WebSocket close frame and exits. We do NOT cancel writeCtx
	// here because that races with draining: Go's select would pick
	// <-ctx.Done() over <-client.Send and drop the error on the floor,
	// which is why clients previously saw "close 1006 (abnormal closure)"
	// instead of the EventError text frame.
	var shutdownOnce sync.Once
	shutdownWritePump := func() {
		shutdownOnce.Do(func() {
			closeClientSend(client)
		})
	}

	// If we return before handing the client to readPump (i.e. during
	// handshake), signal writePump to drain+close and wait for it to
	// finish. After socketReady=true, readPump owns the lifecycle and
	// will shut writePump down via hub.LeaveRoom → closeClientSend.
	socketReady := false
	defer func() {
		if !socketReady {
			shutdownWritePump()
			<-doneWriting
			// writePump has exited — safe to cancel now (also unblocks any
			// ticker-blocked selects if they somehow didn't notice Send
			// close, though writePump is already done at this point).
			writeCancel()
		}
	}()

	// Enqueue auth_ok through the already-running writePump. Any failure
	// to enqueue goes through enqueueError as well (best-effort) before
	// shutdownWritePump closes everything.
	if isGuest {
		if !h.enqueue(client, EventAuthOK, AuthOKPayload{UserID: userID, SessionID: "", ExpiresAt: accessExpiresAt.Unix()}) {
			h.enqueueError(client, errors.WSErrSendFailed, "client send queue unavailable")
			return
		}
	} else {
		if !h.enqueue(client, EventAuthOK, AuthOKPayload{UserID: userID, SessionID: sessionID, ExpiresAt: accessExpiresAt.Unix()}) {
			h.enqueueError(client, errors.WSErrSendFailed, "client send queue unavailable")
			return
		}
	}

	joinPayload, err := h.readJoin(conn)
	if err != nil {
		h.enqueueError(client, errors.WSErrJoinFailed, err.Error())
		return
	}
	if _, err := h.hub.JoinByRequest(r.Context(), joinPayload, client); err != nil {
		h.enqueueError(client, errors.WSErrJoinFailed, err.Error())
		// If JoinByRequest failed AFTER partially registering the client in
		// a room, make sure the room cleans it up. Calling LeaveRoom when
		// the client was never registered is a safe no-op.
		if client.RoomID != "" {
			h.hub.LeaveRoom(client.RoomID, client)
		}
		return
	}

	// Handshake complete — readPump takes over. When readPump returns
	// (disconnect/leave/error) its defer calls hub.LeaveRoom which calls
	// closeClientSend(client), closing client.Send so writePump drains
	// any final messages, sends a Close frame, and exits. We MUST wait
	// for writePump to exit on its own via Send-close before cancelling
	// writeCtx, otherwise writeCancel races the drain just like in the
	// handshake error paths above.
	socketReady = true
	h.readPump(r.Context(), client, authCtx)
	// readPump's LeaveRoom already closed client.Send — writePump is now
	// draining and will exit shortly. Wait for it, then cancel for good
	// measure (the request context is about to die too).
	<-doneWriting
	writeCancel()
}

// readAuth requires the first frame to be an auth envelope. It accepts either
// a registered-user access_token OR a short-lived guest_token (issued by
// POST /rooms/by-code/:code/join). Returns the built AuthContext plus a bool
// reporting whether this is a guest.
func (h *Handler) readAuth(ctx context.Context, conn *websocket.Conn) (*AuthContext, bool, error) {
	_ = conn.SetReadDeadline(time.Now().Add(authDeadline))
	env, err := readEnvelope(conn)
	if err != nil {
		return nil, false, err
	}
	if env.Type != EventAuth {
		return nil, false, fmt.Errorf("first websocket message must be auth")
	}
	var payload AuthPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, false, fmt.Errorf("invalid auth payload")
	}
	if strings.TrimSpace(payload.AccessToken) != "" {
		ac, err := h.authenticator.AuthenticateAccessToken(ctx, payload.AccessToken)
		if err != nil {
			return nil, false, err
		}
		// Best-effort username resolution for the player list. If the lookup
		// fails (DB blip, deleted account mid-connection) we still let the
		// socket through — the client will see the userID until a game_state
		// arrives with a populated Players list (which also looks up names).
		if h.users != nil && ac.Nickname == "" {
			if u, err := h.users.GetByID(ctx, ac.UserID); err == nil && u != nil {
				ac.Nickname = u.Username
			}
		}
		return ac, false, nil
	}
	if strings.TrimSpace(payload.GuestToken) != "" {
		if h.rooms == nil {
			return nil, false, fmt.Errorf("guest auth not available")
		}
		g, err := h.rooms.ValidateGuestToken(ctx, payload.GuestToken)
		if err != nil {
			return nil, false, err
		}
		if g == nil {
			return nil, false, fmt.Errorf("invalid guest token")
		}
		return &AuthContext{
			UserID:          g.GuestID,
			AccessExpiresAt: g.ExpiresAt,
			IsGuest:         true,
			Nickname:        g.Nickname,
			RoomID:          g.RoomID,
		}, true, nil
	}
	return nil, false, fmt.Errorf("access_token or guest_token is required")
}

func (h *Handler) readJoin(conn *websocket.Conn) (JoinPayload, error) {
	_ = conn.SetReadDeadline(time.Now().Add(joinDeadline))
	env, err := readEnvelope(conn)
	if err != nil {
		return JoinPayload{}, err
	}
	if env.Type != EventJoin {
		return JoinPayload{}, fmt.Errorf("second websocket message must be join")
	}
	if len(env.Payload) == 0 {
		// Empty join means: backend, please public-matchmake me.
		return JoinPayload{Mode: "public"}, nil
	}
	var payload JoinPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return JoinPayload{}, fmt.Errorf("invalid join payload")
	}
	payload.RoomID = strings.TrimSpace(payload.RoomID)
	payload.InviteCode = strings.TrimSpace(payload.InviteCode)
	payload.Mode = strings.ToLower(strings.TrimSpace(payload.Mode))
	payload.Language = strings.ToLower(strings.TrimSpace(payload.Language))
	payload.CategoryID = strings.TrimSpace(payload.CategoryID)
	if len(payload.RoomID) > 128 || len(payload.InviteCode) > 64 || len(payload.Language) > 8 || len(payload.CategoryID) > 128 {
		return JoinPayload{}, fmt.Errorf("invalid join payload")
	}
	if payload.Mode != "" && payload.Mode != "public" && payload.Mode != "private" && payload.Mode != "reconnect" {
		return JoinPayload{}, fmt.Errorf("invalid join mode")
	}
	return payload, nil
}

func (h *Handler) readPump(ctx context.Context, client *Client, authCtx *AuthContext) {
	defer func() {
		if client.RoomID != "" {
			h.hub.LeaveRoom(client.RoomID, client)
		}
		// Do not close the connection here: LeaveRoom closes client.Send, then
		// writePump drains any final error/leave messages and closes the socket.
	}()

	_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		return client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	windowStart := time.Now()
	messagesInWindow := 0
	badMessages := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-client.Done:
			return
		default:
		}

		env, err := readEnvelope(client.Conn)
		if err != nil {
			return
		}

		now := time.Now()
		if now.Sub(windowStart) >= time.Second {
			windowStart = now
			messagesInWindow = 0
		}
		messagesInWindow++
		if messagesInWindow > maxMessagesPerSecond {
			h.enqueueError(client, errors.WSErrRateLimited, "too many websocket messages")
			return
		}

		if !authCtx.IsGuest && !h.authenticator.SessionActive(ctx, authCtx) {
			h.enqueueError(client, errors.WSErrSessionRevoked, "session no longer active")
			return
		}
		if authCtx.IsGuest && !authCtx.AccessValid(now) {
			h.enqueueError(client, errors.WSErrAuthExpired, "guest session expired")
			return
		}

		if env.Type == EventAuth {
			if authCtx.IsGuest {
				h.enqueueError(client, errors.WSErrAuthFailed, "guests cannot re-authenticate")
				badMessages++
				if badMessages >= maxConsecutiveBadMessages {
					return
				}
				continue
			}
			if !authCtx.AccessValid(now) {
				h.enqueueError(client, errors.WSErrAuthExpired, "websocket access token expired; reconnect with a fresh access token")
				return
			}
			if err := h.reauthenticate(ctx, client, authCtx, env); err != nil {
				badMessages++
				h.enqueueError(client, errors.WSErrAuthFailed, err.Error())
				if badMessages >= maxConsecutiveBadMessages {
					return
				}
				continue
			}
			badMessages = 0
			continue
		}

		if !authCtx.AccessValid(now) {
			h.enqueueError(client, errors.WSErrAuthExpired, "websocket access token expired; reconnect with a fresh access token")
			return
		}

		if err := validateClientEvent(env); err != nil {
			badMessages++
			h.enqueueError(client, errors.WSErrBadMessage, err.Error())
			if badMessages >= maxConsecutiveBadMessages {
				return
			}
			continue
		}
		badMessages = 0

		if env.Type == EventLeave {
			client.IntentionalLeave = true
			return
		}

		if err := h.hub.DispatchToRoom(client.RoomID, &RoomEvent{
			Type:      env.Type,
			Client:    client,
			Payload:   env.Payload,
			Seq:       env.Seq,
			Timestamp: now,
		}); err != nil {
			h.enqueueError(client, errors.WSErrRoomError, err.Error())
			return
		}
	}
}

func (h *Handler) reauthenticate(ctx context.Context, client *Client, authCtx *AuthContext, env *MessageEnvelope) error {
	var payload AuthPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("invalid auth payload")
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return fmt.Errorf("access token is required")
	}

	next, err := h.authenticator.AuthenticateAccessToken(ctx, payload.AccessToken)
	if err != nil {
		return err
	}
	currentUserID, currentSessionID, _, _ := authCtx.Snapshot()
	nextUserID, nextSessionID, _, nextExpiry := next.Snapshot()
	if nextUserID != currentUserID || nextSessionID != currentSessionID {
		return fmt.Errorf("re-auth token must belong to the same user and session")
	}

	authCtx.UpdateFrom(next)
	returnBool := h.enqueue(client, EventAuthOK, AuthOKPayload{UserID: nextUserID, SessionID: nextSessionID, ExpiresAt: nextExpiry.Unix()})
	if !returnBool {
		return fmt.Errorf("client send queue unavailable")
	}
	return nil
}

func (h *Handler) writePump(ctx context.Context, client *Client, authCtx *AuthContext) {
	pingTicker := time.NewTicker(pingPeriod)
	authTicker := time.NewTicker(sessionCheckPeriod)
	defer func() {
		pingTicker.Stop()
		authTicker.Stop()
		_ = client.Conn.Close()
	}()

	lastAuthRequired := time.Time{}

	for {
		// Check for shutdown (client.Done closed) non-blocking so that
		// once signalled we stop servicing pings/auth-ticks and drain
		// everything pending on client.Send before closing the socket.
		// Without this, a ready ping or auth-ticker case could be chosen
		// over a queued EventError message and the client would see a
		// 1006 abnormal closure instead of the error frame.
		select {
		case <-client.Done:
			h.writePumpDrainAndClose(client)
			return
		default:
		}

		select {
		case <-ctx.Done():
			return
		case <-client.Done:
			h.writePumpDrainAndClose(client)
			return
		case message, ok := <-client.Send:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Send closed without going through Done — still shut
				// down cleanly with a proper close frame.
				_ = client.Conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
					time.Now().Add(writeWait),
				)
				return
			}
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-pingTicker.C:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-authTicker.C:
			if !h.writePumpAuthCheck(ctx, client, authCtx, &lastAuthRequired) {
				return
			}
		}
	}
}

// writePumpDrainAndClose flushes every pending message on client.Send and
// then sends a normal WebSocket close frame. It is invoked when client.Done
// is closed — i.e. after closeClientSend(client) has been called. Because
// closeClientSend closes Send immediately after closing Done, we will see
// ok=false on the next Receive after the buffer empties.
func (h *Handler) writePumpDrainAndClose(client *Client) {
	for {
		select {
		case message, ok := <-client.Send:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = client.Conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
					time.Now().Add(writeWait),
				)
				return
			}
			_ = client.Conn.WriteMessage(websocket.TextMessage, message)
		default:
			// No message ready right now — Send is unbuffered-empty
			// but not yet closed (shouldn't normally happen since
			// closeClientSend closes Send right after Done, but guard
			// against a spin loop just in case).
			time.Sleep(time.Millisecond)
		}
	}
}

// writePumpAuthCheck runs inside writePump so auth/session monitoring does not
// require a third goroutine per socket. This keeps the efficient per-socket model:
// one readPump goroutine and one writePump goroutine.
func (h *Handler) writePumpAuthCheck(ctx context.Context, client *Client, authCtx *AuthContext, lastAuthRequired *time.Time) bool {
	now := time.Now()
	if authCtx.IsGuest {
		if !authCtx.AccessValid(now) {
			_ = h.writeEnvelopeNow(client, EventError, ErrorPayload{Code: errors.WSErrAuthExpired.String(), Message: "guest session expired"})
			if client.RoomID != "" {
				h.hub.LeaveRoom(client.RoomID, client)
			}
			return false
		}
		return true
	}
	if !h.authenticator.SessionActive(ctx, authCtx) {
		_ = h.writeEnvelopeNow(client, EventError, ErrorPayload{Code: errors.WSErrSessionRevoked.String(), Message: "session no longer active"})
		if client.RoomID != "" {
			h.hub.LeaveRoom(client.RoomID, client)
		}
		return false
	}

	accessExpiresAt := authCtx.AccessExpiresAtValue()
	if !authCtx.AccessValid(now) {
		_ = h.writeEnvelopeNow(client, EventError, ErrorPayload{Code: errors.WSErrAuthExpired.String(), Message: "websocket access token expired; reconnect with a fresh access token"})
		if client.RoomID != "" {
			h.hub.LeaveRoom(client.RoomID, client)
		}
		return false
	}

	if now.After(accessExpiresAt.Add(-reauthWarningBefore)) && now.Sub(*lastAuthRequired) >= reauthWarningBefore/2 {
		if err := h.writeEnvelopeNow(client, EventAuthRequired, AuthRequiredPayload{
			ExpiresAt: accessExpiresAt.Unix(),
		}); err != nil {
			return false
		}
		*lastAuthRequired = now
	}
	return true
}

func (h *Handler) writeEnvelopeNow(client *Client, eventType EventType, payload any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	data, err := json.Marshal(MessageEnvelope{Type: eventType, Payload: payloadJSON, Timestamp: time.Now().Unix()})
	if err != nil {
		return err
	}
	_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	return client.Conn.WriteMessage(websocket.TextMessage, data)
}

func readEnvelope(conn *websocket.Conn) (*MessageEnvelope, error) {
	messageType, data, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if messageType != websocket.TextMessage {
		return nil, fmt.Errorf("only text JSON websocket frames are accepted")
	}
	var env MessageEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("invalid JSON envelope")
	}
	if env.Type == "" {
		return nil, fmt.Errorf("message type is required")
	}
	return &env, nil
}

func validateClientEvent(env *MessageEnvelope) error {
	switch env.Type {
	case EventDraw:
		_, err := ValidateDrawingPayload(env.Payload)
		return err
	case EventChat, EventGame, EventClearCanvas:
		if len(env.Payload) == 0 {
			return fmt.Errorf("payload is required")
		}
		if !json.Valid(env.Payload) {
			return fmt.Errorf("payload must be valid JSON")
		}
		return nil
	case EventLeave:
		return nil
	case EventAuth, EventJoin:
		return fmt.Errorf("auth/join are only allowed during handshake")
	default:
		return fmt.Errorf("unsupported websocket event type")
	}
}

func (h *Handler) enqueue(client *Client, eventType EventType, payload any) bool {
	payloadJSON, _ := json.Marshal(payload)
	data, _ := json.Marshal(MessageEnvelope{Type: eventType, Payload: payloadJSON, Timestamp: time.Now().Unix()})
	return safeSend(client, data)
}

func (h *Handler) enqueueError(client *Client, code errors.WSErrorCode, message string) bool {
	return h.enqueue(client, EventError, ErrorPayload{Code: code.String(), Message: message})
}

func writeCloseError(conn *websocket.Conn, code errors.WSErrorCode, message string, closeCode int) {
	payload, _ := json.Marshal(ErrorPayload{Code: code.String(), Message: message})
	data, _ := json.Marshal(MessageEnvelope{Type: EventError, Payload: payload, Timestamp: time.Now().Unix()})
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = conn.WriteMessage(websocket.TextMessage, data)
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, message), time.Now().Add(writeWait))
	_ = conn.Close()
}

func (h *Handler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser clients (tests, CLI tools, internal services) often omit Origin.
		// Browsers always send it, so CSRF-style WebSocket abuse is still blocked.
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	originHost := originURL.Hostname()
	requestHost, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		requestHost = r.Host
	}
	if strings.EqualFold(originHost, requestHost) {
		return true
	}

	if h.cfg.App.Domain != "" {
		appURL, err := url.Parse(h.cfg.App.Domain)
		if err == nil && strings.EqualFold(originHost, appURL.Hostname()) {
			return true
		}
	}
	return false
}
