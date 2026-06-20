package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	App  AppConfig
	Redis RedisConfig
	GRPC GRPCConfig
	JWT  JWTConfig
}

type AppConfig struct {
	Name        string
	Environment string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type GRPCConfig struct {
	Port int
}

type JWTConfig struct {
	PublicKeyPath  string
	AccessTokenTTL int
}

func Load() (*Config, error) {
	redisDB, _ := strconv.Atoi(os.Getenv("CART_REDIS_DB"))
	grpcPort, _ := strconv.Atoi(os.Getenv("CART_GRPC_PORT"))

	cfg := &Config{
		App: AppConfig{
			Name:        "cart-service",
			Environment: "production",
		},
		Redis: RedisConfig{
			Addr:     os.Getenv("CART_REDIS_ADDR"),
			Password: os.Getenv("CART_REDIS_PASSWORD"),
			DB:       redisDB,
		},
		GRPC: GRPCConfig{
			Port: grpcPort,
		},
		JWT: JWTConfig{
			PublicKeyPath: os.Getenv("CART_JWT_PUBLIC_KEY_PATH"),
		},
	}

	if cfg.Redis.Addr == "" {
		return nil, fmt.Errorf("CART_REDIS_ADDR is required")
	}

	return cfg, nil
}
