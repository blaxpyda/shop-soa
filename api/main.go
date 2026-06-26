package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"

	"thugcorp.io/grocery/api/internal/clients"
	"thugcorp.io/grocery/api/internal/handlers"
	"thugcorp.io/grocery/api/internal/middleware"
)

func main() {
	pubKeyBytes, err := os.ReadFile(env("JWT_PUBLIC_KEY_PATH", "../secrets/jwt_public.pem"))
	if err != nil {
		log.Fatalf("read JWT public key: %v", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		log.Fatalf("parse JWT public key: %v", err)
	}

	svc, err := clients.Dial(clients.Targets{
		Auth:          env("AUTH_ADDR", "localhost:50052"),
		Business:      env("BUSINESS_ADDR", "localhost:50053"),
		Catalog:       env("CATALOG_ADDR", "localhost:50054"),
		Ordering:      env("ORDERING_ADDR", "localhost:50055"),
		Payment:       env("PAYMENT_ADDR", "localhost:50056"),
		Notifications: env("NOTIFICATIONS_ADDR", "localhost:50057"),
	})
	if err != nil {
		log.Fatalf("dial services: %v", err)
	}
	defer svc.Close()

	h := handlers.New(svc)
	auth := middleware.Auth(pubKey)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	// ── Public (no JWT) ──────────────────────────────────────────────────────
	r.Post("/v1/auth/signup", h.Signup)
	r.Post("/v1/auth/login", h.Login)
	r.Post("/v1/auth/verify", h.VerifyCode)
	r.Post("/v1/auth/resend", h.ResendCode)

	// ── Protected (JWT required) ──────────────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(auth)

		// Auth / profile
		r.Get("/v1/auth/profile", h.GetProfile)
		r.Put("/v1/auth/profile", h.UpdateProfile)
		r.Put("/v1/auth/password", h.ChangePassword)

		// Products (catalog service)
		r.Get("/v1/products", h.ListProducts)
		r.Get("/v1/products/search", h.SearchProducts)
		r.Get("/v1/products/{id}", h.GetProduct)
		r.Post("/v1/products", h.CreateProduct)
		r.Put("/v1/products/{id}", h.UpdateProduct)
		r.Delete("/v1/products/{id}", h.DeleteProduct)

		// Inventory (catalog service)
		r.Get("/v1/inventory/low-stock-count", h.LowStockCount)
		r.Post("/v1/inventory/availability", h.CheckAvailability)
		r.Get("/v1/inventory/{businessId}", h.ListStock)
		r.Get("/v1/inventory/{businessId}/{productId}", h.GetStock)
		r.Post("/v1/inventory/adjust", h.AdjustStock)

		// Cart (ordering service)
		r.Get("/v1/cart", h.GetCart)
		r.Post("/v1/cart/items", h.AddToCart)
		r.Put("/v1/cart/items/{productId}", h.UpdateCartItem)
		r.Delete("/v1/cart/items/{productId}", h.RemoveFromCart)
		r.Delete("/v1/cart", h.ClearCart)

		// Orders (ordering service)
		r.Post("/v1/orders", h.CreateOrder)
		r.Get("/v1/orders", h.ListOrders)
		r.Get("/v1/orders/{id}", h.GetOrder)
		r.Put("/v1/orders/{id}/status", h.UpdateOrderStatus)
		r.Post("/v1/orders/{id}/cancel", h.CancelOrder)

		// User management (super-admin)
		r.Get("/v1/admin/users", h.ListUsers)
		r.Post("/v1/admin/user", h.CreateUser)
		r.Patch("/v1/admin/users/{userId}/role", h.UpdateUserRole)

		// POS sales
		r.Post("/v1/sales", h.ConfirmSale)

		// Uploads
		r.Post("/v1/uploads/presign", h.PresignUpload)

		// Stats
		r.Get("/v1/stats", h.GetStats)
		r.Get("/v1/my/stats", h.GetMyStats)
		r.Get("/v1/system/stats", h.GetSystemStats)

		// Business
		r.Post("/v1/businesses", h.CreateBusiness)
		r.Get("/v1/businesses/{id}", h.GetBusiness)
		r.Put("/v1/businesses/{id}", h.UpdateBusiness)
		r.Delete("/v1/businesses/{id}", h.DeleteBusiness)
		r.Put("/v1/businesses/{id}/users", h.AddUserToBusiness)

		// Payments (payment service)
		r.Post("/v1/payments", h.InitiatePayment)
		r.Get("/v1/payments/{id}", h.GetPayment)
		r.Post("/v1/payments/{id}/refund", h.Refund)
		r.Get("/v1/payments/balance", h.GetBalance)
		r.Post("/v1/payments/payouts", h.InitiatePayout)

		// Ledger / transactions (payment service)
		r.Get("/v1/transactions", h.ListTransactions)
		r.Get("/v1/transactions/{id}", h.GetTransaction)
		r.Post("/v1/transactions", h.CreateTransaction)
		r.Put("/v1/transactions/{id}/status", h.UpdateTransactionStatus)

		// Notifications
		r.Get("/v1/notifications", h.ListNotifications)
		r.Put("/v1/notifications/read", h.MarkRead)
		r.Get("/v1/notifications/unread-count", h.GetUnreadCount)
		r.Get("/v1/notifications/preferences", h.GetNotificationPreferences)
		r.Put("/v1/notifications/preferences", h.UpdateNotificationPreferences)
	})

	srv := &http.Server{
		Addr:              env("LISTEN_ADDR", ":8090"),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("gateway listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown: %v", err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
