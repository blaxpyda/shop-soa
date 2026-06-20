package middleware

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	st, _ := status.FromError(err)
	if err != nil {
		log.Printf("Method: %s, Duration: %s, Code: %s, Error: %v", info.FullMethod, time.Since(start), st.Code(), err)
	} else {
		log.Printf("Method: %s, Duration: %s, Code: %s", info.FullMethod, time.Since(start), st.Code())
	}
	return resp, err
}
