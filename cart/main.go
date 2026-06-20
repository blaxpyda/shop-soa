package main

import (
	"fmt"
	"log"
	"net"

	"github.com/go-redis/redis/v8"
	pb "thugcorp.io/grocery/cart/proto"

	"os"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"thugcorp.io/grocery/cart/config"
	"thugcorp.io/grocery/cart/internal"
	"thugcorp.io/grocery/cart/internal/middleware"
)

func main() {
	// load configurations
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// initialise dependencies
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	cartRepo := internal.NewCartRepository(redisClient)
	cartService := internal.NewCartService(cartRepo)
	cartHandler := internal.NewCartHandler(cartService)

	// setup interceptor chains
	pubKeyBytes, err := os.ReadFile(cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatalf("failed to read JWT public key: %v", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		log.Fatalf("failed to parse JWT public key: %v", err)
	}

	interceptors := []grpc.UnaryServerInterceptor{
		middleware.LoggingInterceptor,
		middleware.RecoveryInterceptor,
		middleware.AuthInterceptor(pubKey),
	}

	// create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
	)

	// register handlers to the gRPC server
	pb.RegisterCartServiceServer(grpcServer, cartHandler)

	// start listening on the configured port
	addr := fmt.Sprintf(":%d", cfg.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	log.Printf("Starting %s in %s mode on port %d",
		cfg.App.Name,
		cfg.App.Environment,
		cfg.GRPC.Port,
	)

	// start the gRPC server
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
