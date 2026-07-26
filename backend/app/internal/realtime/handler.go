package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
	"drawo/pkg/errors"
)

const (
	authDeadline              = 5 * time.Second
	joinDeadline              = 10 * time.Second
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

type Handler struct {
	cfg           config.Config
	hub           *Hub
	authenticator *Authenticator
	upgrader      websocket.Upgrader
}

func NewHandler(cfg config.Config, hub *Hub, sessions repositories.SessionRepository) *Handler {
	h := &Handler{
		cfg:           cfg,
		hub:           hub,
		authenticator: NewAuthenticator(cfg, sessions),
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

	authCtx, err := h.readAuth(r.Context(), conn)
	if err != nil {
		writeCloseError(conn, errors.WSErrAuthFailed, err.Error(), websocket.ClosePolicyViolation)
		return
	}

	userID, sessionID, _, accessExpiresAt := authCtx.Snapshot()
	client := &Client{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan []byte, clientSendBuffer),
		Done:      make(chan struct{}),
	}

	if !h.enqueue(client, EventAuthOK, AuthOKPayload{UserID: userID, SessionID: sessionID, ExpiresAt: accessExpiresAt.Unix()}) {
		writeCloseError(conn, errors.WSErrSendFailed, "client send queue unavailable", websocket.CloseInternalServerErr)
		return
	}

	joinPayload, err := h.readJoin(conn)
	if err != nil {
		writeCloseError(conn, errors.WSErrJoinFailed, err.Error(), websocket.ClosePolicyViolation)
		return
	}
	if _, err := h.hub.JoinByRequest(r.Context(), joinPayload, client); err != nil {
		writeCloseError(conn, errors.WSErrJoinFailed, err.Error(), websocket.ClosePolicyViolation)
		return
	}

	go h.writePump(r.Context(), client, authCtx)
	h.readPump(r.Context(), client, authCtx)
}

// readAuth requires the first frame to be an auth envelope. This avoids putting
// access tokens in query strings, where reverse proxies and logs commonly leak
// them. Browser clients can send this frame immediately after WebSocket open.
func (h *Handler) readAuth(ctx context.Context, conn *websocket.Conn) (*AuthContext, error) {
	_ = conn.SetReadDeadline(time.Now().Add(authDeadline))
	env, err := readEnvelope(conn)
	if err != nil {
		return nil, err
	}
	if env.Type != EventAuth {
		return nil, fmt.Errorf("first websocket message must be auth")
	}
	var payload AuthPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid auth payload")
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return nil, fmt.Errorf("access token is required")
	}
	return h.authenticator.AuthenticateAccessToken(ctx, payload.AccessToken)
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

		if !h.authenticator.SessionActive(ctx, authCtx) {
			h.enqueueError(client, errors.WSErrSessionRevoked, "session no longer active")
			return
		}

		if env.Type == EventAuth {
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
		select {
		case <-ctx.Done():
			return
		case message, ok := <-client.Send:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
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

// writePumpAuthCheck runs inside writePump so auth/session monitoring does not
// require a third goroutine per socket. This keeps the efficient per-socket model:
// one readPump goroutine and one writePump goroutine.
func (h *Handler) writePumpAuthCheck(ctx context.Context, client *Client, authCtx *AuthContext, lastAuthRequired *time.Time) bool {
	now := time.Now()
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
