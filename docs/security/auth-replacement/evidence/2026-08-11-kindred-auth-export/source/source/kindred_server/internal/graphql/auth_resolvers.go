package graphql

import (
	"context"
	"errors"
	"time"

	"kindred_server/internal/analytics"
	"kindred_server/internal/auth"
	apperrors "kindred_server/internal/platform/errors"
	"kindred_server/internal/points"
	"kindred_server/internal/stats"
	"kindred_server/internal/subscription"
	"kindred_server/internal/user"
)

type AuthService interface {
	Register(context.Context, auth.RegisterRequest) (auth.AuthResponse, error)
	Login(context.Context, auth.LoginRequest) (auth.AuthResponse, error)
	Reactivate(context.Context, auth.ReactivateRequest) (auth.AuthResponse, error)
	Refresh(context.Context, auth.RefreshRequest) (auth.AuthResponse, error)
	Logout(context.Context, auth.LogoutRequest) error
	VerifyEmail(context.Context, auth.VerifyEmailRequest) error
	ResendVerification(context.Context, string) error
	RequestPasswordReset(context.Context, string) error
	ResetPassword(context.Context, auth.ResetPasswordRequest) error
	StartPhoneVerification(context.Context, string) (auth.StartPhoneVerificationResponse, error)
	VerifyPhone(context.Context, string, auth.VerifyPhoneRequest) error
	StartPhoneChange(context.Context, string, auth.StartPhoneChangeRequest) (auth.StartPhoneVerificationResponse, error)
	VerifyPhoneChange(context.Context, string, auth.VerifyPhoneChangeRequest) (user.Response, error)
}

type UserLookup interface {
	GetByID(context.Context, string) (user.User, error)
}

type MeStatsReader interface {
	Me(context.Context, string) (stats.MeStats, error)
}

type PointsReader interface {
	GetBalance(context.Context, string) (points.Balance, error)
	ListLedger(context.Context, string, int, string) ([]points.Transaction, string, error)
}

type SubscriptionStatusReader interface {
	Status(context.Context, string, time.Time) (subscription.Status, error)
}

type ConsentStateReader interface {
	CurrentConsents(context.Context, string) (analytics.ConsentStateResponse, error)
}

type AuthResolverOption func(*authResolverConfig)

type authResolverConfig struct {
	stats            MeStatsReader
	points           PointsReader
	subscription     SubscriptionStatusReader
	consents         ConsentStateReader
	storeKitEnv      string
	settingsPageSize int
	now              func() time.Time
}

func WithViewerStats(reader MeStatsReader) AuthResolverOption {
	return func(cfg *authResolverConfig) {
		cfg.stats = reader
	}
}

func WithViewerPoints(reader PointsReader) AuthResolverOption {
	return func(cfg *authResolverConfig) {
		cfg.points = reader
	}
}

func WithViewerSubscription(reader SubscriptionStatusReader, storeKitEnv string) AuthResolverOption {
	return func(cfg *authResolverConfig) {
		cfg.subscription = reader
		cfg.storeKitEnv = storeKitEnv
	}
}

func WithViewerConsents(reader ConsentStateReader) AuthResolverOption {
	return func(cfg *authResolverConfig) {
		cfg.consents = reader
	}
}

func WithViewerClock(now func() time.Time) AuthResolverOption {
	return func(cfg *authResolverConfig) {
		cfg.now = now
	}
}

func RegisterAuthResolvers(handler *Handler, authService AuthService, users UserLookup, opts ...AuthResolverOption) {
	cfg := authResolverConfig{
		settingsPageSize: 20,
		now:              func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	handler.Register("Mutation", "register", PublicAPIKeyResolver(func(ctx context.Context, event ResolverEvent) (any, error) {
		input, err := DecodeInput[auth.RegisterRequest](event)
		if err != nil {
			return nil, err
		}
		return authService.Register(ctx, input)
	}))
	handler.Register("Mutation", "login", PublicAPIKeyResolver(func(ctx context.Context, event ResolverEvent) (any, error) {
		input, err := DecodeInput[auth.LoginRequest](event)
		if err != nil {
			return nil, err
		}
		return authService.Login(ctx, input)
	}))
	handler.Register("Mutation", "reactivate", PublicAPIKeyResolver(func(ctx context.Context, event ResolverEvent) (any, error) {
		input, err := DecodeInput[auth.ReactivateRequest](event)
		if err != nil {
			return nil, err
		}
		return authService.Reactivate(ctx, input)
	}))
	handler.Register("Mutation", "refreshSession", PublicAPIKeyResolver(func(ctx context.Context, event ResolverEvent) (any, error) {
		input, err := DecodeInput[auth.RefreshRequest](event)
		if err != nil {
			return nil, err
		}
		return authService.Refresh(ctx, input)
	}))
	handler.Register("Mutation", "logout", ProtectedResolver(func(ctx context.Context, event ResolverEvent, claims auth.Claims) (any, error) {
		if _, err := activeUserForClaims(ctx, users, claims); err != nil {
			return nil, err
		}
		input, err := DecodeInput[auth.LogoutRequest](event)
		if err != nil {
			return nil, err
		}
		return true, authService.Logout(ctx, input)
	}))
	handler.Register("Mutation", "verifyEmail", PublicAPIKeyResolver(func(ctx context.Context, event ResolverEvent) (any, error) {
		input, err := DecodeInput[auth.VerifyEmailRequest](event)
		if err != nil {
			return nil, err
		}
		return true, authService.VerifyEmail(ctx, input)
	}))
	handler.Register("Mutation", "resendEmailVerification", PublicAPIKeyResolver(func(ctx context.Context, event ResolverEvent) (any, error) {
		input, err := DecodeInput[auth.ResendVerificationRequest](event)
		if err != nil {
			return nil, err
		}
		return true, authService.ResendVerification(ctx, input.Email)
	}))
	handler.Register("Mutation", "forgotPassword", PublicAPIKeyResolver(func(ctx context.Context, event ResolverEvent) (any, error) {
		input, err := DecodeInput[auth.RequestPasswordResetRequest](event)
		if err != nil {
			return nil, err
		}
		return true, authService.RequestPasswordReset(ctx, input.Email)
	}))
	handler.Register("Mutation", "resetPassword", PublicAPIKeyResolver(func(ctx context.Context, event ResolverEvent) (any, error) {
		input, err := DecodeInput[auth.ResetPasswordRequest](event)
		if err != nil {
			return nil, err
		}
		return true, authService.ResetPassword(ctx, input)
	}))

	handler.Register("Query", "me", ProtectedResolver(func(ctx context.Context, event ResolverEvent, claims auth.Claims) (any, error) {
		found, err := activeUserForClaims(ctx, users, claims)
		if err != nil {
			return nil, err
		}
		return buildViewerPayload(ctx, found, cfg)
	}))

	handler.Register("Mutation", "startPhoneVerification", ProtectedResolver(func(ctx context.Context, event ResolverEvent, claims auth.Claims) (any, error) {
		found, err := activeUserForClaims(ctx, users, claims)
		if err != nil {
			return nil, err
		}
		return authService.StartPhoneVerification(ctx, found.ID)
	}))
	handler.Register("Mutation", "verifyPhone", ProtectedResolver(func(ctx context.Context, event ResolverEvent, claims auth.Claims) (any, error) {
		found, err := activeUserForClaims(ctx, users, claims)
		if err != nil {
			return nil, err
		}
		input, err := DecodeInput[auth.VerifyPhoneRequest](event)
		if err != nil {
			return nil, err
		}
		return true, authService.VerifyPhone(ctx, found.ID, input)
	}))
	handler.Register("Mutation", "startPhoneChange", ProtectedResolver(func(ctx context.Context, event ResolverEvent, claims auth.Claims) (any, error) {
		found, err := activeUserForClaims(ctx, users, claims)
		if err != nil {
			return nil, err
		}
		input, err := DecodeInput[auth.StartPhoneChangeRequest](event)
		if err != nil {
			return nil, err
		}
		return authService.StartPhoneChange(ctx, found.ID, input)
	}))
	handler.Register("Mutation", "verifyPhoneChange", ProtectedResolver(func(ctx context.Context, event ResolverEvent, claims auth.Claims) (any, error) {
		found, err := activeUserForClaims(ctx, users, claims)
		if err != nil {
			return nil, err
		}
		input, err := DecodeInput[auth.VerifyPhoneChangeRequest](event)
		if err != nil {
			return nil, err
		}
		return authService.VerifyPhoneChange(ctx, found.ID, input)
	}))
}

func activeUserForClaims(ctx context.Context, users UserLookup, claims auth.Claims) (user.User, error) {
	found, err := users.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return user.User{}, auth.ErrInvalidToken
		}
		return user.User{}, apperrors.Internal(err)
	}
	if found.Status() != user.AccountStatusActive || found.TokenVersion != claims.TokenVersion {
		return user.User{}, auth.ErrInvalidToken
	}
	return found, nil
}

type viewerPayload struct {
	ID              string                    `json:"id"`
	Email           string                    `json:"email,omitempty"`
	DisplayName     string                    `json:"displayName,omitempty"`
	Phone           string                    `json:"phone,omitempty"`
	PhoneVerified   bool                      `json:"phoneVerified"`
	EmailVerified   bool                      `json:"emailVerified"`
	City            string                    `json:"city,omitempty"`
	BirthYear       int                       `json:"birthYear,omitempty"`
	Gender          string                    `json:"gender,omitempty"`
	ProfilePhotoURL string                    `json:"profilePhotoUrl,omitempty"`
	Stats           viewerStatsPayload        `json:"stats"`
	Points          viewerPointsPayload       `json:"points"`
	Subscription    subscriptionEntitlement   `json:"subscription"`
	Consents        viewerConsentStatePayload `json:"consents"`
	CreatedAt       time.Time                 `json:"createdAt,omitempty"`
	UpdatedAt       time.Time                 `json:"updatedAt,omitempty"`
	AccountStatus   user.AccountStatus        `json:"accountStatus,omitempty"`
}

type viewerStatsPayload struct {
	PointsAvailable     int    `json:"pointsAvailable"`
	ItemsListed         int    `json:"itemsListed"`
	ItemsActive         int    `json:"itemsActive"`
	ItemsCompleted      int    `json:"itemsCompleted"`
	RequestsOpen        int    `json:"requestsOpen"`
	RequestsCompleted   int    `json:"requestsCompleted"`
	ActiveRequests      int    `json:"activeRequests"`
	CompletedDeliveries int    `json:"completedDeliveries"`
	City                string `json:"city"`
}

type viewerPointsPayload struct {
	Balance    points.Balance       `json:"balance"`
	Ledger     []points.Transaction `json:"ledger"`
	NextCursor *string              `json:"nextCursor,omitempty"`
}

type subscriptionEntitlement struct {
	Active                          bool       `json:"active"`
	ProductID                       string     `json:"productId,omitempty"`
	ExpiresAt                       *time.Time `json:"expiresAt,omitempty"`
	Environment                     string     `json:"environment,omitempty"`
	FreeEarlyRequestsRemainingToday int        `json:"freeEarlyRequestsRemainingToday"`
	DailyFreeEarlyRequestLimit      int        `json:"dailyFreeEarlyRequestLimit"`
}

func subscriptionEntitlementFromStatus(status subscription.Status, storeKitEnv string) subscriptionEntitlement {
	return subscriptionEntitlement{
		Active:                          status.IsActive,
		ProductID:                       status.ProductID,
		ExpiresAt:                       status.ExpiresAt,
		Environment:                     storeKitEnv,
		FreeEarlyRequestsRemainingToday: status.FreeEarlyRequestsRemainingToday,
		DailyFreeEarlyRequestLimit:      status.DailyFreeEarlyRequestLimit,
	}
}

type viewerConsentStatePayload struct {
	Consents any `json:"consents"`
}

func buildViewerPayload(ctx context.Context, found user.User, cfg authResolverConfig) (viewerPayload, error) {
	base := user.ToResponse(found)
	payload := viewerPayload{
		ID:            base.ID,
		Email:         base.Email,
		DisplayName:   base.DisplayName,
		Phone:         base.Phone,
		PhoneVerified: base.PhoneVerified,
		EmailVerified: base.EmailVerified,
		City:          base.City,
		BirthYear:     base.BirthYear,
		Gender:        base.Gender,
		CreatedAt:     base.CreatedAt,
		UpdatedAt:     base.UpdatedAt,
		AccountStatus: base.AccountStatus,
		Stats: viewerStatsPayload{
			City: base.City,
		},
		Points: viewerPointsPayload{
			Balance: points.Balance{UserID: base.ID},
			Ledger:  []points.Transaction{},
		},
		Subscription: subscriptionEntitlement{
			Environment: cfg.storeKitEnv,
		},
		Consents: viewerConsentStatePayload{
			Consents: map[string]any{},
		},
	}

	if cfg.stats != nil {
		meStats, err := cfg.stats.Me(ctx, base.ID)
		if err != nil {
			return viewerPayload{}, err
		}
		payload.Stats = viewerStatsPayload{
			PointsAvailable:     meStats.PointsAvailable,
			ItemsListed:         meStats.ItemsListed,
			ItemsActive:         meStats.ItemsActive,
			ItemsCompleted:      meStats.ItemsCompleted,
			RequestsOpen:        meStats.RequestsOpen,
			RequestsCompleted:   meStats.RequestsCompleted,
			ActiveRequests:      meStats.RequestsOpen,
			CompletedDeliveries: meStats.ItemsCompleted,
			City:                meStats.City,
		}
	}

	if cfg.points != nil {
		balance, err := cfg.points.GetBalance(ctx, base.ID)
		if err != nil {
			return viewerPayload{}, apperrors.Internal(err)
		}
		ledger, nextCursor, err := cfg.points.ListLedger(ctx, base.ID, cfg.settingsPageSize, "")
		if err != nil {
			return viewerPayload{}, apperrors.Internal(err)
		}
		payload.Points = viewerPointsPayload{
			Balance:    balance,
			Ledger:     ledger,
			NextCursor: stringPtr(nextCursor),
		}
	}

	if cfg.subscription != nil {
		status, err := cfg.subscription.Status(ctx, base.ID, cfg.now())
		if err != nil {
			return viewerPayload{}, err
		}
		payload.Subscription = subscriptionEntitlementFromStatus(status, cfg.storeKitEnv)
	}

	if cfg.consents != nil {
		state, err := cfg.consents.CurrentConsents(ctx, base.ID)
		if err != nil {
			return viewerPayload{}, err
		}
		payload.Consents = viewerConsentStatePayload{Consents: state.Consents}
	}

	return payload, nil
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
