package config

type Config struct {
	App    App
	Server Server
	DB     DB
}

type App struct {
	Name      string
	SecretKey string
}

type Server struct {
	Host string
	Port string
}

type DB struct {
	Name string
	User string
	Pass string
	Host string
	Port string
}
