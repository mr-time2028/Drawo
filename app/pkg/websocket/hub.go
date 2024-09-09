package websocket

var h *Hub

type Hub struct {
	Rooms      map[string]*Room
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message
}

func NewHub() {
	h = &Hub{
		Rooms:      make(map[string]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *Message),
	}
}

func GetHub() *Hub {
	return h
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			// check room exists
			if _, ok := h.Rooms[client.RoomID]; ok {
				// add client to the room
				r := h.Rooms[client.RoomID] // specify which room the client request to add to it

				if r.Clients == nil {
					// Initialize the Clients map because a client want to join to this room
					r.Clients = make(map[string]*Client)
				}

				if _, ok = r.Clients[client.ID]; !ok {
					r.Clients[client.ID] = client // add client to the room if client not in the room
				}
			}
		case client := <-h.Unregister:
			// check if room exists
			if _, ok := h.Rooms[client.RoomID]; ok {
				// delete client from room (should search in all rooms in hub and find client)
				if _, ok = h.Rooms[client.RoomID].Clients[client.ID]; ok {
					delete(h.Rooms[client.RoomID].Clients, client.ID)
					close(client.Message)

					// broadcast a message that the client has left the room
					if len(h.Rooms[client.RoomID].Clients) != 0 {
						h.Broadcast <- &Message{
							//Content:  fmt.Sprintf("user %s left the chat", client.Username),
							//SenderID: client.Username,
							//RoomID:   client.RoomID,
						}
					}

					// Set Clients map to nil if it's empty
					if len(h.Rooms[client.RoomID].Clients) == 0 {
						h.Rooms[client.RoomID].Clients = nil
					}
				}
			}
		case message := <-h.Broadcast:
			// check if room exists
			if _, ok := h.Rooms[message.RoomID]; ok {
				// broadcast message to all clients in the room
				for _, client := range h.Rooms[message.RoomID].Clients {
					client.Message <- message
				}
			}
		}
	}
}
