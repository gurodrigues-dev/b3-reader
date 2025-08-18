package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	FilePath    string `mapstructure:"FILE_PATH"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`
	ServerPort  string `mapstructure:"SERVER_PORT"`
}

func LoadEnvs() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("fail to load .env: %v", err)
	}

	viper.AutomaticEnv()

	viper.SetDefault("DATABASE_URL", "")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("FILE_PATH", "")
	viper.SetDefault("SERVER_PORT", "8083")

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
