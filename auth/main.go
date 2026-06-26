package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
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

	db, err := gorm.Open(sqlite.Open(cfg.SQLLite.Path), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&domain.User{}); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	bootstrapSuperAdmin(db, cfg.SuperAdmin)

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

	sms := utils.NewEasySendSMS(cfg.SMS.APIKey, cfg.SMS.SenderID)
	email := utils.NewEmailService(cfg.Gmail.Host, cfg.Gmail.Port, cfg.Gmail.Username, cfg.Gmail.Password)

	authRepo := internal.NewAuthRepository(db)
	authService := internal.NewAuthService(authRepo, sms, email, privKey, cfg.JWT.AccessTokenTTL)
	authHandler := internal.NewAuthHandler(authService)

	interceptors := []grpc.UnaryServerInterceptor{
		middleware.LoggingInterceptor,
		middleware.RecoveryInterceptor,
		middleware.AuthInterceptor(pubKey),
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	pb.RegisterIdentityServiceServer(grpcServer, authHandler)

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

func bootstrapSuperAdmin(db *gorm.DB, cfg config.SuperAdminConfig) {
	if cfg.Email == "" || cfg.Password == "" {
		log.Println("AUTH_SUPER_ADMIN_EMAIL/PASSWORD not set — skipping super admin bootstrap")
		return
	}

	// Check by email, not by role constant — the constant value may change during
	// refactors and would otherwise trigger a duplicate-bootstrap crash.
	var count int64
	db.Model(&domain.User{}).Where("email = ?", cfg.Email).Count(&count)
	if count > 0 {
		// Ensure the existing account still carries the super-admin role in case
		// a previous bootstrap ran under a different constant value (e.g. "superadmin").
		db.Model(&domain.User{}).Where("email = ?", cfg.Email).Update("role", domain.RoleSuperAdmin)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(cfg.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bootstrap: bcrypt: %v", err)
	}

	var id string
	if err := db.Raw("SELECT lower(hex(randomblob(16)))").Scan(&id).Error; err != nil {
		log.Fatalf("bootstrap: id generation: %v", err)
	}

	admin := domain.User{
		ID:         id,
		Email:      cfg.Email,
		Password:   string(hashed),
		FirstName:  cfg.FirstName,
		LastName:   cfg.LastName,
		Role:       domain.RoleSuperAdmin,
		IsVerified: true,
	}
	if err := db.WithContext(context.Background()).Create(&admin).Error; err != nil {
		log.Fatalf("bootstrap: create super admin: %v", err)
	}
	log.Printf("super admin bootstrapped: %s", cfg.Email)
}
