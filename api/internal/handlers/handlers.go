package handlers

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
	"thugcorp.io/grocery/api/internal/clients"
	"thugcorp.io/grocery/api/internal/middleware"
)

// Handlers holds all downstream gRPC clients.
// Each domain lives in its own file (auth.go, products.go, …).
type Handlers struct {
	svc *clients.Services
}

func New(svc *clients.Services) *Handlers {
	return &Handlers{svc: svc}
}

// outgoingCtx injects the caller's Bearer token into outgoing gRPC metadata.
// Call this inside any protected handler before making a gRPC request so that
// downstream services can validate the JWT independently.
func (h *Handlers) outgoingCtx(r *http.Request) context.Context {
	_, _, _, token := middleware.ClaimsFromCtx(r.Context())
	return metadata.AppendToOutgoingContext(r.Context(), "authorization", "Bearer "+token)
}
