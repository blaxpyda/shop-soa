package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App    AppConfig    `mapstructure:"app"`
	SQLite SQLiteConfig `mapstructure:"sqlite"`
	GRPC   GRPCConfig   `mapstructure:"grpc"`
	JWT    JWTConfig    `mapstructure:"jwt"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
}

type SQLiteConfig struct {
	Path string `mapstructure:"path"`
}

type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

type JWTConfig struct {
	PublicKeyPath string `mapstructure:"public_key_path"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	_ = godotenv.Load()

	viper.SetEnvPrefix("CATALOG")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.SQLite.Path == "" {
		return nil, fmt.Errorf("sqlite.path is required")
	}

	return &cfg, nil
}
