package config

import (
	"github.com/spf13/viper"
)

func ParseConfig() error {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	return err
}
