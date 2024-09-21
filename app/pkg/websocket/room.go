package websocket

type Room struct {
	ID           string
	Name         string
	IdentifierID string
	Password     string
	Clients      map[string]*Client
}
