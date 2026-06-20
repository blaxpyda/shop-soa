package middleware

import (
	"context"
	"crypto/rsa"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	UserIDKey     contextKey = "userID"
	BusinessIDKey contextKey = "businessID"
	RoleKey       contextKey = "role"
)

type CustomClaims struct {
	UserID     string `json:"user_id"`
	BusinessID string `json:"business_id"`
	Role       string `json:"role"`
	jwt.RegisteredClaims
}

func AuthInterceptor(publicKey *rsa.PublicKey) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if info.FullMethod == "/proto.CartService/PublicMethod" {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
		}

		tokenString := strings.TrimPrefix(authHeaders[0], "Bearer ")
		claims, err := validateToken(tokenString, publicKey)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, BusinessIDKey, claims.BusinessID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)
		return handler(ctx, req)
	}
}

func validateToken(tokenString string, publicKey *rsa.PublicKey) (*CustomClaims, error) {
	if tokenString == "" {
		return nil, status.Errorf(codes.Unauthenticated, "missing token")
	}

	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error){
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, status.Errorf(codes.Unauthenticated, "unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	if !token.Valid {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token")
	}

	if claims.UserID == "" && claims.BusinessID == "" && claims.Role == "" {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: missing user or business or role ID")
	}

	return claims, nil
}
