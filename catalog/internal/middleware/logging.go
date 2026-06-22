package middleware

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	st, _ := status.FromError(err)
	if err != nil {
		log.Printf("method=%s duration=%s code=%s error=%v", info.FullMethod, duration, st.Code(), err)
	} else {
		log.Printf("method=%s duration=%s code=%s", info.FullMethod, duration, st.Code())
	}
	return resp, err
}
