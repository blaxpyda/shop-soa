package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	"thugcorp.io/catalog/config"
	"thugcorp.io/catalog/internal"
	"thugcorp.io/catalog/internal/domain"
	"thugcorp.io/catalog/internal/middleware"
	pb "thugcorp.io/catalog/proto"
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
		&domain.Product{},
		&domain.StockItem{},
		&domain.Reservation{},
		&domain.ReservationItem{},
		&domain.AdjustmentLog{},
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

	repo := internal.NewCatalogRepository(db)
	svc := internal.NewCatalogService(repo)
	handler := internal.NewCatalogHandler(svc)

	interceptors := []grpc.UnaryServerInterceptor{
		middleware.LoggingInterceptor,
		middleware.RecoveryInterceptor,
		middleware.AuthInterceptor(pubKey),
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	pb.RegisterCatalogServiceServer(grpcServer, handler)

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
