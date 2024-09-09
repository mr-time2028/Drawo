package websocket

type Room struct {
	ID         string
	Name       string
	Identifier string
	Password   string
	Clients    map[string]*Client
}
