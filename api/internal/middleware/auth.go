package middleware

import (
	"context"
	"crypto/rsa"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"thugcorp.io/grocery/api/internal/respond"
)

type contextKey string

const (
	UserIDKey     contextKey = "userID"
	RoleKey       contextKey = "role"
	BusinessIDKey contextKey = "businessID"
	TokenKey      contextKey = "rawToken" // forwarded to downstream gRPC calls
)

type claims struct {
	UserID     string `json:"user_id"`
	Role       string `json:"role"`
	BusinessID string `json:"business_id"`
	jwt.RegisteredClaims
}

// Auth validates a Bearer JWT (RS256) signed by the auth service.
// On success it injects UserID, Role, BusinessID, and the raw token into ctx.
func Auth(publicKey *rsa.PublicKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				respond.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			if tokenStr == header {
				respond.Error(w, http.StatusUnauthorized, "authorization header must be Bearer token")
				return
			}

			c := &claims{}
			token, err := jwt.ParseWithClaims(tokenStr, c, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return publicKey, nil
			})
			if err != nil || !token.Valid {
				respond.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, c.UserID)
			ctx = context.WithValue(ctx, RoleKey, c.Role)
			ctx = context.WithValue(ctx, BusinessIDKey, c.BusinessID)
			ctx = context.WithValue(ctx, TokenKey, tokenStr)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromCtx is a convenience used by handlers to read the injected identity.
func ClaimsFromCtx(ctx context.Context) (userID, role, businessID, token string) {
	userID, _ = ctx.Value(UserIDKey).(string)
	role, _ = ctx.Value(RoleKey).(string)
	businessID, _ = ctx.Value(BusinessIDKey).(string)
	token, _ = ctx.Value(TokenKey).(string)
	return
}
