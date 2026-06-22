package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	"thugcorp.io/payment/config"
	"thugcorp.io/payment/internal"
	"thugcorp.io/payment/internal/domain"
	"thugcorp.io/payment/internal/middleware"
	pb "thugcorp.io/payment/proto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.SQLite.Path), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(
		&domain.Payment{},
		&domain.Transaction{},
		&domain.Payout{},
	); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	pubKeyBytes, err := os.ReadFile(cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatalf("failed to read JWT public key: %v", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		log.Fatalf("failed to parse JWT public key: %v", err)
	}

	repo := internal.NewPaymentRepository(db)
	svc := internal.NewPaymentService(repo)
	handler := internal.NewPaymentHandler(svc)

	interceptors := []grpc.UnaryServerInterceptor{
		middleware.LoggingInterceptor,
		middleware.RecoveryInterceptor,
		middleware.AuthInterceptor(pubKey),
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	pb.RegisterPaymentServiceServer(grpcServer, handler)

	addr := fmt.Sprintf(":%d", cfg.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/payment", internal.WebhookHandler(svc))

	webhookAddr := ":8081"
	webhookServer := &http.Server{
		Addr:    webhookAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("starting webhook server on %s", webhookAddr)
		if err := webhookServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("webhook server error: %v", err)
		}
	}()

	log.Printf("starting %s in %s mode on gRPC port %d",
		cfg.App.Name,
		cfg.App.Environment,
		cfg.GRPC.Port,
	)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
