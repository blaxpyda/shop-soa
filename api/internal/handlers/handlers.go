package handlers

import (
	"context"
	"net/http"
	"strconv"

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

func pageSize(r *http.Request) int32 {
	n, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if n <= 0 {
		return 20
	}
	return int32(n)
}

// outgoingCtx injects the caller's Bearer token into outgoing gRPC metadata.
// Call this inside any protected handler before making a gRPC request so that
// downstream services can validate the JWT independently.
func (h *Handlers) outgoingCtx(r *http.Request) context.Context {
	_, _, _, token := middleware.ClaimsFromCtx(r.Context())
	return metadata.AppendToOutgoingContext(r.Context(), "authorization", "Bearer "+token)
}
