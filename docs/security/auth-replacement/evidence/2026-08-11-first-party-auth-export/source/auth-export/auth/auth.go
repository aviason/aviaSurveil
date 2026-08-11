package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrUnauthorized = errors.New("unauthorized")
var ErrStepUpRequired = errors.New("step_up_required")

type Config struct {
	TokenVerifier    *TokenService
	AllowDevHeader   bool
	SessionValidator SessionValidator
}

type User struct {
	ID           int64
	Role         string
	Subject      string
	SessionID    string
	ClientID     string
	TokenID      string
	AuthTime     time.Time
	AuthVersion  int64
	LastStepUpAt *time.Time
}

type contextKey struct{}

type SessionValidator interface {
	ValidateSession(ctx context.Context, user User) (User, bool, error)
}

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(contextKey{}).(User)
	return user, ok
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	user, ok := UserFromContext(ctx)
	if !ok || user.ID <= 0 {
		return 0, false
	}
	return user.ID, true
}

func HasRecentStepUp(ctx context.Context, ttl time.Duration, now time.Time) bool {
	user, ok := UserFromContext(ctx)
	if !ok {
		return false
	}
	return UserHasRecentStepUp(user, ttl, now)
}

func UserHasRecentStepUp(user User, ttl time.Duration, now time.Time) bool {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return user.LastStepUpAt != nil && now.UTC().Sub(user.LastStepUpAt.UTC()) <= ttl
}

func Wrap(next http.Handler, cfg Config, requiresAuth func(*http.Request) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresAuth(r) {
			next.ServeHTTP(w, withOptionalUser(r, cfg))
			return
		}

		user, err := Authenticate(r, cfg)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if cfg.SessionValidator != nil && shouldValidateSession(user) {
			validated, active, err := cfg.SessionValidator.ValidateSession(r.Context(), user)
			if err != nil || !active {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			user = validated
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

func withOptionalUser(r *http.Request, cfg Config) *http.Request {
	user, err := Authenticate(r, cfg)
	if err != nil {
		return r
	}
	if cfg.SessionValidator != nil && shouldValidateSession(user) {
		validated, active, err := cfg.SessionValidator.ValidateSession(r.Context(), user)
		if err != nil || !active {
			return r
		}
		user = validated
	}
	return r.WithContext(WithUser(r.Context(), user))
}

func Authenticate(r *http.Request, cfg Config) (User, error) {
	if cfg.AllowDevHeader {
		if value := strings.TrimSpace(r.Header.Get("X-User-ID")); value != "" {
			userID, err := strconv.ParseInt(value, 10, 64)
			if err != nil || userID <= 0 {
				return User{}, ErrUnauthorized
			}
			return User{ID: userID, Role: strings.TrimSpace(r.Header.Get("X-User-Role"))}, nil
		}
	}

	const prefix = "Bearer "
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, prefix) || cfg.TokenVerifier == nil {
		return User{}, ErrUnauthorized
	}
	return cfg.TokenVerifier.VerifyAccessToken(strings.TrimSpace(strings.TrimPrefix(authHeader, prefix)), time.Now().UTC())
}

func shouldValidateSession(user User) bool {
	return user.SessionID != "" || user.Subject != ""
}

func claimUserID(claims map[string]any) (int64, bool) {
	for _, key := range []string{"userId", "uid", "user_id"} {
		if value, ok := int64Claim(claims[key]); ok {
			return value, true
		}
	}
	subject, ok := claims["sub"].(string)
	if !ok {
		return 0, false
	}
	subject = strings.TrimPrefix(subject, "user-")
	value, err := strconv.ParseInt(subject, 10, 64)
	return value, err == nil
}

func int64Claim(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	case float64:
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
