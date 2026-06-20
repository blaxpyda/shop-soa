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
	"thugcorp.io/grocery/transaction/config"
	"thugcorp.io/grocery/transaction/internal"
	"thugcorp.io/grocery/transaction/internal/domain"
	"thugcorp.io/grocery/transaction/internal/middleware"
	pb "thugcorp.io/grocery/transaction/proto"
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

	if err := db.AutoMigrate(
		&domain.Transaction{},
		&domain.TransactionItem{},
	); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	repo := internal.NewTransactionRepository(db)
	svc := internal.NewTransactionService(
		repo,
	)
	handler := internal.NewTransactionHandler(svc)

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

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))
	pb.RegisterTransactionServiceServer(grpcServer, handler)

	addr := fmt.Sprintf(":%d", cfg.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	log.Printf("starting %s in %s mode on gRPC port %d",
		cfg.App.Name, cfg.App.Environment, cfg.GRPC.Port)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func connectPostgres(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User,
		cfg.Postgres.Password, cfg.Postgres.DBName, cfg.Postgres.SSLMode,
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
