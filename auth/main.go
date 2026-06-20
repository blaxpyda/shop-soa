package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"thugcorp.io/grocery/auth/config"
	"thugcorp.io/grocery/auth/internal"
	"thugcorp.io/grocery/auth/internal/domain"
	"thugcorp.io/grocery/auth/internal/middleware"
	"thugcorp.io/grocery/auth/internal/utils"
	pb "thugcorp.io/grocery/auth/proto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := connectPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	if err := db.AutoMigrate(&domain.User{}); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	privKeyBytes, err := os.ReadFile(cfg.JWT.PrivateKeyPath)
	if err != nil {
		log.Fatalf("failed to read JWT private key: %v", err)
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyBytes)
	if err != nil {
		log.Fatalf("failed to parse JWT private key: %v", err)
	}

	pubKeyBytes, err := os.ReadFile(cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatalf("failed to read JWT public key: %v", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		log.Fatalf("failed to parse JWT public key: %v", err)
	}

	sms := utils.NewEasySendSMS(os.Getenv("EASYSEND_API_KEY"), os.Getenv("EASYSEND_SENDER"))

	authRepo := internal.NewAuthRepository(db)
	authService := internal.NewAuthService(authRepo, sms, privKey, cfg.JWT.AccessTokenTTL)
	authHandler := internal.NewAuthHandler(authService)

	interceptors := []grpc.UnaryServerInterceptor{
		middleware.LoggingInterceptor,
		middleware.RecoveryInterceptor,
		middleware.AuthInterceptor(pubKey),
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	pb.RegisterAuthServiceServer(grpcServer, authHandler)

	addr := fmt.Sprintf(":%d", cfg.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	log.Printf("starting %s in %s mode on gRPC port %d",
		cfg.App.Name,
		cfg.App.Environment,
		cfg.GRPC.Port,
	)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func connectPostgres(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.SSLMode,
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
