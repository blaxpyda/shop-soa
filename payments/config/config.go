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
	MTN      MTNConfig
}

type MTNConfig struct {
	BaseURL         string
	SubscriptionKey string
	APIUser         string
	APIKey          string
	Environment     string
	PayeeID         string
}

type AppConfig struct {
	Name        string
	Environment string
	Port        int
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
	pgPort, _ := strconv.Atoi(os.Getenv("PAYMENT_POSTGRES_PORT"))
	grpcPort, _ := strconv.Atoi(os.Getenv("PAYMENT_GRPC_PORT"))

	cfg := &Config{
		App: AppConfig{
			Name:        "payment-service",
			Environment: "production",
		},
		Postgres: PostgresConfig{
			Host:     os.Getenv("PAYMENT_POSTGRES_HOST"),
			Port:     pgPort,
			User:     os.Getenv("PAYMENT_POSTGRES_USER"),
			Password: os.Getenv("PAYMENT_POSTGRES_PASSWORD"),
			DBName:   os.Getenv("PAYMENT_POSTGRES_DBNAME"),
			SSLMode:  os.Getenv("PAYMENT_POSTGRES_SSLMODE"),
		},
		GRPC: GRPCConfig{
			Port: grpcPort,
		},
		JWT: JWTConfig{
			PublicKeyPath: os.Getenv("PAYMENT_JWT_PUBLIC_KEY_PATH"),
		},
		MTN: MTNConfig{
			BaseURL:         os.Getenv("PAYMENT_MTN_BASE_URL"),
			SubscriptionKey: os.Getenv("PAYMENT_MTN_SUBSCRIPTION_KEY"),
			APIUser:         os.Getenv("PAYMENT_MTN_API_USER"),
			APIKey:          os.Getenv("PAYMENT_MTN_API_KEY"),
			Environment:     os.Getenv("PAYMENT_MTN_ENVIRONMENT"),
			PayeeID:         os.Getenv("PAYMENT_MTN_PAYEE_ID"),
		},
	}

	if cfg.Postgres.Host == "" {
		return nil, fmt.Errorf("PAYMENT_POSTGRES_HOST is required")
	}

	return cfg, nil
}
