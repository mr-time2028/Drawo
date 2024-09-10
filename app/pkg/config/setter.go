package config

import (
	"drawo/config"
	"github.com/spf13/viper"
	"log"
)

var configurations config.Config

func SetConfig() {
	// defaults
	viper.SetDefault("app.name", "Drawo")

	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", "8080")

	viper.SetDefault("db.host", "localhost")
	viper.SetDefault("db.port", "5432")

	viper.SetDefault("auth.issuer", "localhost")
	viper.SetDefault("auth.audience", "localhost")
	viper.SetDefault("auth.tokenExpiry", 5)
	viper.SetDefault("auth.refreshExpiry", 60)

	// set configs
	viper.SetConfigName(".config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")

	// read configs
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("Error reading the configs")
	}

	err := viper.Unmarshal(&configurations)
	if err != nil {
		log.Fatal("Unable to decode configs")
	}
}
