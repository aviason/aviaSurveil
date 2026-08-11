package middleware

import (
	"context"
	"net/http"
	"strings"

	"kindred_server/internal/auth"
	"kindred_server/internal/platform/errors"
	"kindred_server/internal/platform/router"
	"kindred_server/pkg/response"
)

type contextKey string

const userIDKey contextKey = "authenticated_user_id"

func Authenticator(service *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || strings.TrimSpace(token) == "" {
				response.Error(w, auth.ErrInvalidToken)
				return
			}
			claims, err := service.VerifyToken(r.Context(), token)
			if err != nil {
				response.Error(w, err)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(r *http.Request) string {
	id, _ := r.Context().Value(userIDKey).(string)
	return id
}

// RequireOwnership ensures the authenticated user can only access their own resources
func RequireOwnership(paramName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserID(r)
			if userID == "" {
				response.Error(w, auth.ErrInvalidToken)
				return
			}

			resourceID := router.Param(r, paramName)
			if userID != resourceID {
				response.Error(w, errors.Forbidden("cannot access other users' resources"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
