package config

import (
	"drawo/config"
	"github.com/spf13/viper"
	"log"
)

var configurations config.Config

func SetConfig() {
	viper.SetConfigName(".config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("Error reading the configs")
	}

	err := viper.Unmarshal(&configurations)
	if err != nil {
		log.Fatal("Unable to decode configs")
	}
}
