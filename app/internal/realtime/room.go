package realtime

import (
	"context"
	"encoding/json"
	"time"

	"drawo/internal/core/domain"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 64 * 1024 // 64 KiB hard limit to prevent memory abuse.
)

type Room struct {
	state   *domain.Room
	clients map[string]*Client
	inbox   chan *RoomEvent
	onClose func(roomID, inviteCode string)
}

func NewRoom(state *domain.Room, onClose func(string, string)) *Room {
	return &Room{
		state:   state,
		clients: make(map[string]*Client),
		inbox:   make(chan *RoomEvent, 512),
		onClose: onClose,
	}
}

// Dispatch queues an event for this room's single goroutine.
// It is intentionally non-blocking: if a room cannot keep up, callers receive
// false and may disconnect/back-pressure the client rather than deadlocking the
// WebSocket read pump.
func (r *Room) Dispatch(e *RoomEvent) bool {
	select {
	case r.inbox <- e:
		return true
	default:
		return false
	}
}

func (r *Room) Run(ctx context.Context) {
	defer func() {
		for _, client := range r.clients {
			closeClientSend(client)
		}
		r.onClose(r.state.ID, r.state.InviteCode)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-r.inbox:
			if !r.handleEvent(event) {
				return
			}
		}
	}
}

func (r *Room) handleEvent(e *RoomEvent) bool {
	if e == nil || e.Client == nil {
		return true
	}

	switch e.Type {
	case EventJoin:
		r.clients[e.Client.ID] = e.Client
		r.sendSystem(e.Client, EventJoined, map[string]string{"room_id": r.state.ID})
		r.broadcast(e, e.Client.ID)
	case EventLeave:
		if client, ok := r.clients[e.Client.ID]; ok {
			delete(r.clients, e.Client.ID)
			closeClientSend(client)
		}
		r.broadcast(e, e.Client.ID)
		if len(r.clients) == 0 && r.state.State == domain.RoomStatePlaying {
			return false // Close active ephemeral game rooms once everyone leaves.
		}
	case EventChat, EventDraw, EventGame, EventClearCanvas, EventGameState:
		if _, ok := r.clients[e.Client.ID]; ok {
			r.broadcast(e, "")
		}
	}
	return true
}

func (r *Room) broadcast(e *RoomEvent, excludeClientID string) {
	data, err := json.Marshal(MessageEnvelope{
		Type:      e.Type,
		Payload:   e.Payload,
		Seq:       e.Seq,
		Timestamp: e.Timestamp.Unix(),
	})
	if err != nil {
		return
	}

	for _, client := range r.clients {
		if client.ID == excludeClientID {
			continue
		}
		if !safeSend(client, data) {
			delete(r.clients, client.ID)
			closeClientSend(client)
		}
	}
}

func (r *Room) sendSystem(client *Client, eventType EventType, payload any) {
	payloadJSON, _ := json.Marshal(payload)
	data, _ := json.Marshal(MessageEnvelope{
		Type:      eventType,
		Payload:   payloadJSON,
		Timestamp: time.Now().Unix(),
	})
	if !safeSend(client, data) {
		delete(r.clients, client.ID)
		closeClientSend(client)
	}
}

func safeSend(client *Client, data []byte) bool {
	select {
	case client.Send <- data:
		return true
	default:
		return false
	}
}

func closeClientSend(client *Client) {
	if client == nil {
		return
	}
	if client.Done == nil {
		if client.Send != nil {
			close(client.Send)
		}
		return
	}
	select {
	case <-client.Done:
		// Already closed.
	default:
		close(client.Done)
		if client.Send != nil {
			close(client.Send)
		}
	}
}
