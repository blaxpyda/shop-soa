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
	"thugcorp.io/ordering/config"
	"thugcorp.io/ordering/internal"
	"thugcorp.io/ordering/internal/clients"
	"thugcorp.io/ordering/internal/domain"
	"thugcorp.io/ordering/internal/middleware"
	pb "thugcorp.io/ordering/proto"
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
		&domain.Cart{},
		&domain.CartItem{},
		&domain.Order{},
		&domain.OrderItem{},
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

	catalogClient, catalogConn, err := clients.NewCatalogClient(cfg.Catalog.Address)
	if err != nil {
		log.Fatalf("failed to connect to catalog service at %s: %v", cfg.Catalog.Address, err)
	}
	defer catalogConn.Close()

	repo := internal.NewOrderingRepository(db)
	svc := internal.NewOrderingService(repo, catalogClient)
	handler := internal.NewOrderingHandler(svc)

	interceptors := []grpc.UnaryServerInterceptor{
		middleware.LoggingInterceptor,
		middleware.RecoveryInterceptor,
		middleware.AuthInterceptor(pubKey),
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	pb.RegisterOrderingServiceServer(grpcServer, handler)

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
