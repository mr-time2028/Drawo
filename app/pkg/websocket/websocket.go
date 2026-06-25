package websocket

import (
	"github.com/gorilla/websocket"
	"net/http"
)

var (
	UpgradeConnection = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type WsAuthPayload struct {
	Token string `json:"token"`
}

type WsRequest struct {
	//Token       string  `json:"token"`
	Action        string        `json:"action"`
	WsAuthPayload WsAuthPayload `json:"auth"`
	//Message     string  `json:"message"`
	//MessageType string  `json:"message_type"`
	Client *Client `json:"-"`
}

type WsResponse struct {
	Error          bool     `json:"error"`
	Status         int      `json:"status"`
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	MessageType    string   `json:"message_type"`
	ConnectedUsers []string `json:"connected_users"`
}
