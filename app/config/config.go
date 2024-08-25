package config

import "time"

type Config struct {
	App    App
	Server Server
	DB     DB
	Auth   Auth
}

type App struct {
	Name      string
	Domain    string
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

type Auth struct {
	Issuer        string
	Audience      string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}
