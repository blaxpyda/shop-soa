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
	startTime := time.Now()

	resp, err := handler(ctx, req)
	duration := time.Since(startTime)

	st, _ := status.FromError(err)
	code := st.Code().String()

	if err != nil {
		log.Printf("Method: %s, Duration: %s, Code: %s, Error: %v", info.FullMethod, duration, code, err)
	} else {
		log.Printf("Method: %s, Duration: %s, Code: %s", info.FullMethod, duration, code)
	}

	return resp, err
}
