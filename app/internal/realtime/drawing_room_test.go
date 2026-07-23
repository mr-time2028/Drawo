package realtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/internal/core/domain"
)

func testRoomForDrawing(currentDrawerID string) (*Room, *Client, *Client) {
	room := NewRoom(&domain.Room{ID: "draw-room", State: domain.RoomStatePlaying, CurrentDrawerID: currentDrawerID}, func(string, string) {}, nil, nil, nil, nil)
	drawer := &Client{ID: "drawer-conn", UserID: "drawer", Send: make(chan []byte, 50), Done: make(chan struct{})}
	guesser := &Client{ID: "guesser-conn", UserID: "guesser", Send: make(chan []byte, 50), Done: make(chan struct{})}
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: drawer, Timestamp: time.Now()})
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: guesser, Timestamp: time.Now()})
	room.gameState = GameStateDrawing
	drainClient(drawer)
	drainClient(guesser)
	return room, drawer, guesser
}

func drainClient(client *Client) {
	for len(client.Send) > 0 {
		<-client.Send
	}
}

func nextEnvelope(t *testing.T, client *Client) MessageEnvelope {
	t.Helper()
	select {
	case data := <-client.Send:
		var env MessageEnvelope
		require.NoError(t, json.Unmarshal(data, &env))
		return env
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket envelope")
	}
	return MessageEnvelope{}
}

func strokePayload(color string) json.RawMessage {
	payload, _ := json.Marshal(DrawOperation{
		Op:     DrawOpStroke,
		Tool:   ToolPencil,
		Color:  color,
		Size:   4,
		Points: []Point{{X: 1, Y: 1}, {X: 2, Y: 2}},
	})
	return payload
}

func TestRoomDrawingHistoryUndoRedoAndCanvasSync(t *testing.T) {
	room, drawer, guesser := testRoomForDrawing("drawer")

	room.handleEvent(&RoomEvent{Type: EventDraw, Client: drawer, Payload: strokePayload("#000000"), Timestamp: time.Now()})
	drawMsg := nextEnvelope(t, guesser)
	assert.Equal(t, EventDraw, drawMsg.Type)
	assert.Len(t, room.canvasOps, 1)
	assert.Equal(t, int64(1), room.canvasOps[0].ServerSeq)
	assert.NotEmpty(t, room.canvasOps[0].ID)

	room.handleEvent(&RoomEvent{Type: EventDraw, Client: drawer, Payload: json.RawMessage(`{"op":"undo"}`), Timestamp: time.Now()})
	undoMsg := nextEnvelope(t, guesser)
	assert.Equal(t, EventDraw, undoMsg.Type)
	assert.Len(t, room.canvasOps, 0)
	assert.Len(t, room.redoOps[drawer.UserID], 1)

	room.handleEvent(&RoomEvent{Type: EventDraw, Client: drawer, Payload: json.RawMessage(`{"op":"redo"}`), Timestamp: time.Now()})
	redoMsg := nextEnvelope(t, guesser)
	assert.Equal(t, EventDraw, redoMsg.Type)
	assert.Len(t, room.canvasOps, 1)
	assert.Len(t, room.redoOps[drawer.UserID], 0)

	newJoiner := &Client{ID: "new-conn", UserID: "new-user", Send: make(chan []byte, 10), Done: make(chan struct{})}
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: newJoiner, Timestamp: time.Now()})
	syncMsg := nextEnvelope(t, newJoiner)
	assert.Equal(t, EventCanvasSync, syncMsg.Type)
	var syncPayload CanvasSyncPayload
	require.NoError(t, json.Unmarshal(syncMsg.Payload, &syncPayload))
	assert.Len(t, syncPayload.Operations, 1)
	assert.Equal(t, room.drawSeq, syncPayload.ServerSeq)
}

func TestRoomDrawingRejectsInvalidDrawAndNonDrawer(t *testing.T) {
	room, drawer, _ := testRoomForDrawing("drawer")

	room.handleEvent(&RoomEvent{Type: EventDraw, Client: drawer, Payload: json.RawMessage(`{"op":"stroke","tool":"pencil","color":"bad","size":4,"points":[{"x":1,"y":1},{"x":2,"y":2}]}`), Timestamp: time.Now()})
	msg := nextEnvelope(t, drawer)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "invalid_draw")

	intruder := &Client{ID: "intruder-conn", UserID: "intruder", Send: make(chan []byte, 10), Done: make(chan struct{})}
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: intruder, Timestamp: time.Now()})
	drainClient(intruder)
	room.handleEvent(&RoomEvent{Type: EventDraw, Client: intruder, Payload: strokePayload("#000000"), Timestamp: time.Now()})
	msg = nextEnvelope(t, intruder)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "draw_forbidden")
}

func TestRoomDrawingRateLimitsClearSpam(t *testing.T) {
	room, drawer, _ := testRoomForDrawing("drawer")

	for i := 0; i < maxClearOpsPerMinute; i++ {
		room.handleEvent(&RoomEvent{Type: EventDraw, Client: drawer, Payload: json.RawMessage(`{"op":"clear"}`), Timestamp: time.Now()})
	}
	drainClient(drawer)

	room.handleEvent(&RoomEvent{Type: EventDraw, Client: drawer, Payload: json.RawMessage(`{"op":"clear"}`), Timestamp: time.Now()})
	msg := nextEnvelope(t, drawer)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "draw_rate_limited")
}
