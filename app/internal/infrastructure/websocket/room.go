package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"drawo/internal/core/domain"
	"drawo/pkg/logger"
)

// Room represents an active, ephemeral, in-memory game room running on this server instance.
//
// DESIGN DECISION:
//   Each active room runs its own dedicated goroutine (`Run()`) processing messages serially
//   from a buffered `inbox` channel. This guarantees thread-safety without mutex contention
//   on hot paths like drawing stroke synchronization or chat broadcasts.
//
//   Rooms are ephemeral runtime entities: when the game finishes or the last player leaves,
//   the room goroutine terminates itself and triggers `onClose`, which removes it from the Hub
//   and deletes discovery entries and invite codes from distributed cache (Redis).
type Room struct {
	ID         string
	InviteCode string
	inbox      chan *RoomEvent
	clients    map[string]*Client
	state      *domain.Room
	onClose    func(roomID, inviteCode string)
	closed     chan struct{}
}

// NewRoom constructs a new ephemeral runtime room goroutine container.
func NewRoom(state *domain.Room, onClose func(roomID, inviteCode string)) *Room {
	return &Room{
		ID:         state.ID,
		InviteCode: state.InviteCode,
		inbox:      make(chan *RoomEvent, 256), // Buffered to absorb drawing bursts
		clients:    make(map[string]*Client),
		state:      state,
		onClose:    onClose,
		closed:     make(chan struct{}),
	}
}

// Run executes the serial event loop for this ephemeral room goroutine.
func (r *Room) Run(ctx context.Context) {
	logger.L.Info("starting ephemeral room goroutine", slog.String("room_id", r.ID), slog.String("invite_code", r.InviteCode))
	defer func() {
		logger.L.Info("destroying ephemeral room goroutine", slog.String("room_id", r.ID))
		close(r.closed)
		if r.onClose != nil {
			r.onClose(r.ID, r.InviteCode)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-r.inbox:
			if !ok {
				return
			}
			r.handleEvent(event)
			// Check lifecycle rule: destroy room when all players leave after initial join
			if len(r.clients) == 0 && r.state.State != domain.RoomStateLobby {
				return
			}
		}
	}
}

// handleEvent processes incoming events inside the single room goroutine.
func (r *Room) handleEvent(event *RoomEvent) {
	switch event.Type {
	case EventJoin:
		if event.Client != nil {
			r.clients[event.Client.ID] = event.Client
			r.broadcastEnvelope(EventJoin, map[string]string{
				"client_id": event.Client.ID,
				"username":  event.Client.Username,
			})
		}
	case EventLeave:
		if event.Client != nil {
			if _, exists := r.clients[event.Client.ID]; exists {
				delete(r.clients, event.Client.ID)
				close(event.Client.Send)
				r.broadcastEnvelope(EventLeave, map[string]string{
					"client_id": event.Client.ID,
				})
			}
			if len(r.clients) == 0 {
				r.state.State = domain.RoomStateClosed
			}
		}
	case EventDraw, EventClearCanvas, EventChat:
		// Echo drawing/chat events to all other players in the ephemeral room
		r.broadcastRaw(event.Type, event.Payload, event.Client)
	}
}

// Dispatch sends a structured event to the room's inbox without blocking.
func (r *Room) Dispatch(event *RoomEvent) {
	select {
	case r.inbox <- event:
	default:
		logger.L.Warn("room inbox full; dropping event", slog.String("room_id", r.ID))
	}
}

// broadcastEnvelope sends a structured JSON payload to all connected clients.
func (r *Room) broadcastEnvelope(eventType EventType, payload interface{}) {
	env := MessageEnvelope{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
	}
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	for _, c := range r.clients {
		select {
		case c.Send <- data:
		default:
			// Client send buffer full; disconnect lagging client
			close(c.Send)
			delete(r.clients, c.ID)
		}
	}
}

// broadcastRaw broadcasts raw payload bytes to all clients except the sender.
func (r *Room) broadcastRaw(eventType EventType, payload []byte, sender *Client) {
	env := MessageEnvelope{
		Type:      eventType,
		Payload:   json.RawMessage(payload),
		Timestamp: time.Now().UnixMilli(),
	}
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	for _, c := range r.clients {
		if sender != nil && c.ID == sender.ID {
			continue
		}
		select {
		case c.Send <- data:
		default:
			close(c.Send)
			delete(r.clients, c.ID)
		}
	}
}

// Close stops the room goroutine cleanly.
func (r *Room) Close() {
	select {
	case <-r.closed:
	default:
		close(r.inbox)
	}
}
