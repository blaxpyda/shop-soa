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
	Name        string
	Environment string
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
	PublicKeyPath string
}

func Load() (*Config, error) {
	pgPort, _ := strconv.Atoi(os.Getenv("INVENTORY_POSTGRES_PORT"))
	grpcPort, _ := strconv.Atoi(os.Getenv("INVENTORY_GRPC_PORT"))

	cfg := &Config{
		App: AppConfig{
			Name:        "inventory-service",
			Environment: "production",
		},
		Postgres: PostgresConfig{
			Host:     os.Getenv("INVENTORY_POSTGRES_HOST"),
			Port:     pgPort,
			User:     os.Getenv("INVENTORY_POSTGRES_USER"),
			Password: os.Getenv("INVENTORY_POSTGRES_PASSWORD"),
			DBName:   os.Getenv("INVENTORY_POSTGRES_DBNAME"),
			SSLMode:  os.Getenv("INVENTORY_POSTGRES_SSLMODE"),
		},
		GRPC: GRPCConfig{
			Port: grpcPort,
		},
		JWT: JWTConfig{
			PublicKeyPath: os.Getenv("INVENTORY_JWT_PUBLIC_KEY_PATH"),
		},
	}

	if cfg.Postgres.Host == "" {
		return nil, fmt.Errorf("INVENTORY_POSTGRES_HOST is required")
	}

	return cfg, nil
}
