package main

import (
	"fmt"
	"log"
	"net"

	"os"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"thugcorp.io/grocery/business/config"
	"thugcorp.io/grocery/business/internal"
	"thugcorp.io/grocery/business/internal/domain"
	"thugcorp.io/grocery/business/internal/middleware"
	pb "thugcorp.io/grocery/business/proto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.SQLLite.Path), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&domain.Business{}); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	repo := internal.NewBusinessRepository(db)
	svc := internal.NewBusinessService(repo)
	handler := internal.NewBusinessHandler(svc)

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

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	pb.RegisterBusinessServiceServer(grpcServer, handler)

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
