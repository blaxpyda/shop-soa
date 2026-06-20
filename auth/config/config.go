package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	GRPC     GRPCConfig
	JWT      JWTConfig
}

type AppConfig struct {
	Name         string
	Environment  string
	Port         int
	LogLevel     string
	ReadTimeout  int
	WriteTimeout int
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type GRPCConfig struct {
	Port int
}

type JWTConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
	AccessTokenTTL int
}

func Load() (*Config, error) {
	pgPort, _ := strconv.Atoi(os.Getenv("AUTH_POSTGRES_PORT"))
	grpcPort, _ := strconv.Atoi(os.Getenv("AUTH_GRPC_PORT"))

	cfg := &Config{
		App: AppConfig{
			Name:        "auth-service",
			Environment: "production",
		},
		Postgres: PostgresConfig{
			Host:     os.Getenv("AUTH_POSTGRES_HOST"),
			Port:     pgPort,
			User:     os.Getenv("AUTH_POSTGRES_USER"),
			Password: os.Getenv("AUTH_POSTGRES_PASSWORD"),
			DBName:   os.Getenv("AUTH_POSTGRES_DBNAME"),
			SSLMode:  os.Getenv("AUTH_POSTGRES_SSLMODE"),
		},
		GRPC: GRPCConfig{
			Port: grpcPort,
		},
		JWT: JWTConfig{
			PrivateKeyPath: os.Getenv("AUTH_JWT_PRIVATE_KEY_PATH"),
			PublicKeyPath:  os.Getenv("AUTH_JWT_PUBLIC_KEY_PATH"),
		},
	}

	if cfg.Postgres.Host == "" {
		return nil, fmt.Errorf("AUTH_POSTGRES_HOST is required")
	}

	return cfg, nil
}
