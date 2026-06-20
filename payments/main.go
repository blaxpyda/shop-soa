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

	"thugcorp.io/grocery/payment/config"
	"thugcorp.io/grocery/payment/internal"
	"thugcorp.io/grocery/payment/internal/domain"
	"thugcorp.io/grocery/payment/internal/middleware"
	"thugcorp.io/grocery/payment/internal/mtn"
	pb "thugcorp.io/grocery/payment/proto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := connectPostgres(cfg)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	if err := db.AutoMigrate(&domain.Payment{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	pubKeyBytes, err := os.ReadFile(cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatalf("read JWT public key: %v", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		log.Fatalf("parse JWT public key: %v", err)
	}

	repo := internal.NewPaymentRepository(db)
	mtnClient := mtn.NewClient(
		cfg.MTN.BaseURL,
		cfg.MTN.SubscriptionKey,
		cfg.MTN.APIUser,
		cfg.MTN.APIKey,
		cfg.MTN.Environment,
		cfg.MTN.PayeeID,
	)
	svc := internal.NewPaymentService(repo, mtnClient)
	handler := internal.NewPaymentHandler(svc)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.LoggingInterceptor,
			middleware.RecoveryInterceptor,
			middleware.AuthInterceptor(pubKey),
		),
	)
	pb.RegisterPaymentServiceServer(grpcServer, handler)

	addr := fmt.Sprintf(":%d", cfg.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen on %s: %v", addr, err)
	}

	log.Printf("starting %s in %s mode on gRPC port %d",
		cfg.App.Name, cfg.App.Environment, cfg.GRPC.Port)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
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
