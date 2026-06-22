package config

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type App struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
}

type SQLite struct {
	Path string `mapstructure:"path"`
}

type GRPC struct {
	Port int `mapstructure:"port"`
}

type JWT struct {
	PublicKeyPath string `mapstructure:"public_key_path"`
}

type Config struct {
	App    App    `mapstructure:"app"`
	SQLite SQLite `mapstructure:"sqlite"`
	GRPC   GRPC   `mapstructure:"grpc"`
	JWT    JWT    `mapstructure:"jwt"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("PAYMENT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
