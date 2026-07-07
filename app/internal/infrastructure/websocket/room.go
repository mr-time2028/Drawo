package websocket

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
	maxMessageSize = 1024 * 4
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
		inbox:   make(chan *RoomEvent, 256),
		onClose: onClose,
	}
}

func (r *Room) Dispatch(e *RoomEvent) {
	r.inbox <- e
}

func (r *Room) Run(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
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
		case <-ticker.C:
			// Ping/Heartbeat logic
		}
	}
}

func (r *Room) handleEvent(e *RoomEvent) bool {
	switch e.Type {
	case EventJoin:
		r.clients[e.Client.ID] = e.Client
	case EventLeave:
		delete(r.clients, e.Client.ID)
		if len(r.clients) == 0 && r.state.State == domain.RoomStatePlaying {
			return false // Signal loop to exit
		}
	case EventDraw:
		r.broadcast(e)
	}
	return true
}

func (r *Room) broadcast(e *RoomEvent) {
	data, _ := json.Marshal(MessageEnvelope{
		Type:      e.Type,
		Payload:   e.Payload,
		Timestamp: e.Timestamp.Unix(),
	})
	for _, client := range r.clients {
		select {
		case client.Send <- data:
		default:
			delete(r.clients, client.ID)
		}
	}
}
