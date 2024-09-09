package websocket

import (
	"github.com/gorilla/websocket"
	"log"
)

type Client struct {
	ID       string
	Username string
	RoomID   string
	Conn     *websocket.Conn
	Hub      *Hub
	Message  chan *Message
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{}
}

func (c *Client) ReadMessage() {
	defer func() {
		c.Hub.Unregister <- c
		_ = c.Conn.Close()
	}()

	for {
		_, m, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		msg := &Message{
			Content: string(m),
			RoomID:  c.RoomID,
		}
		c.Hub.Broadcast <- msg
	}
}

func (c *Client) WriteMessage() {
	defer func() {
		c.Hub.Unregister <- c
		_ = c.Conn.Close()
	}()

	for {
		message, ok := <-c.Message
		if !ok {
			return
		}

		_ = c.Conn.WriteJSON(message)
	}
}
