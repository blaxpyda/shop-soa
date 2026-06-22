package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	SQLLite SQLLiteConfig `mapstructure:"sqlite"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Port        int    `mapstructure:"port"`
	LogLevel    string `mapstructure:"log_level"`
}

type SQLLiteConfig struct {
	Path string `mapstructure:"path"`
}

type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

type JWTConfig struct {
	PrivateKeyPath string `mapstructure:"private_key_path"`
	PublicKeyPath  string `mapstructure:"public_key_path"`
	AccessTokenTTL int    `mapstructure:"access_token_ttl"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	_ = godotenv.Load()

	viper.SetEnvPrefix("NOTIFICATIONS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.SQLLite.Path == "" {
		return nil, fmt.Errorf("sqlite.path is required")
	}

	return &cfg, nil
}
