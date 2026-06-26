package middleware

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// ForwardAuth copies the "authorization" header from the incoming gRPC
// metadata into the outgoing metadata so downstream service-to-service
// calls carry the original caller's JWT.
func ForwardAuth(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", authHeaders[0]))
}
