package graphql_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"kindred_server/internal/analytics"
	"kindred_server/internal/auth"
	"kindred_server/internal/graphql"
	apperrors "kindred_server/internal/platform/errors"
	"kindred_server/internal/points"
	"kindred_server/internal/stats"
	"kindred_server/internal/subscription"
	"kindred_server/internal/user"
)

type fakeAuthService struct {
	registerRequest             auth.RegisterRequest
	loginRequest                auth.LoginRequest
	refreshRequest              auth.RefreshRequest
	logoutRequest               auth.LogoutRequest
	startPhoneVerificationUser  string
	verifyPhoneUser             string
	verifyPhoneRequest          auth.VerifyPhoneRequest
	startPhoneChangeUser        string
	startPhoneChangeRequest     auth.StartPhoneChangeRequest
	verifyPhoneChangeUser       string
	verifyPhoneChangeRequest    auth.VerifyPhoneChangeRequest
	authResponse                auth.AuthResponse
	phoneVerificationResponse   auth.StartPhoneVerificationResponse
	verifyPhoneChangeUserOutput user.Response
}

func (f *fakeAuthService) Register(_ context.Context, req auth.RegisterRequest) (auth.AuthResponse, error) {
	f.registerRequest = req
	return f.authResponse, nil
}

func (f *fakeAuthService) Login(_ context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
	f.loginRequest = req
	return f.authResponse, nil
}

func (f *fakeAuthService) Reactivate(context.Context, auth.ReactivateRequest) (auth.AuthResponse, error) {
	return f.authResponse, nil
}

func (f *fakeAuthService) Refresh(_ context.Context, req auth.RefreshRequest) (auth.AuthResponse, error) {
	f.refreshRequest = req
	return f.authResponse, nil
}

func (f *fakeAuthService) Logout(_ context.Context, req auth.LogoutRequest) error {
	f.logoutRequest = req
	return nil
}

func (f *fakeAuthService) VerifyEmail(context.Context, auth.VerifyEmailRequest) error {
	return nil
}

func (f *fakeAuthService) ResendVerification(context.Context, string) error {
	return nil
}

func (f *fakeAuthService) RequestPasswordReset(context.Context, string) error {
	return nil
}

func (f *fakeAuthService) ResetPassword(context.Context, auth.ResetPasswordRequest) error {
	return nil
}

func (f *fakeAuthService) StartPhoneVerification(_ context.Context, userID string) (auth.StartPhoneVerificationResponse, error) {
	f.startPhoneVerificationUser = userID
	return f.phoneVerificationResponse, nil
}

func (f *fakeAuthService) VerifyPhone(_ context.Context, userID string, req auth.VerifyPhoneRequest) error {
	f.verifyPhoneUser = userID
	f.verifyPhoneRequest = req
	return nil
}

func (f *fakeAuthService) StartPhoneChange(_ context.Context, userID string, req auth.StartPhoneChangeRequest) (auth.StartPhoneVerificationResponse, error) {
	f.startPhoneChangeUser = userID
	f.startPhoneChangeRequest = req
	return f.phoneVerificationResponse, nil
}

func (f *fakeAuthService) VerifyPhoneChange(_ context.Context, userID string, req auth.VerifyPhoneChangeRequest) (user.Response, error) {
	f.verifyPhoneChangeUser = userID
	f.verifyPhoneChangeRequest = req
	return f.verifyPhoneChangeUserOutput, nil
}

type fakeUserLookup struct {
	user user.User
	err  error
}

func (f fakeUserLookup) GetByID(context.Context, string) (user.User, error) {
	return f.user, f.err
}

type fakeMeStatsReader struct {
	stats stats.MeStats
	err   error
}

func (f fakeMeStatsReader) Me(context.Context, string) (stats.MeStats, error) {
	return f.stats, f.err
}

type fakePointsReader struct {
	balance    points.Balance
	ledger     []points.Transaction
	nextCursor string
	err        error
}

func (f fakePointsReader) GetBalance(context.Context, string) (points.Balance, error) {
	return f.balance, f.err
}

func (f fakePointsReader) ListLedger(context.Context, string, int, string) ([]points.Transaction, string, error) {
	return f.ledger, f.nextCursor, f.err
}

type fakeSubscriptionReader struct {
	status subscription.Status
	err    error
}

func (f fakeSubscriptionReader) Status(context.Context, string, time.Time) (subscription.Status, error) {
	return f.status, f.err
}

type fakeConsentReader struct {
	state analytics.ConsentStateResponse
	err   error
}

func (f fakeConsentReader) CurrentConsents(context.Context, string) (analytics.ConsentStateResponse, error) {
	return f.state, f.err
}

func TestRegisterAuthResolversRegisterDecodesInputAndCallsService(t *testing.T) {
	authService := &fakeAuthService{
		authResponse: auth.AuthResponse{Token: "token"},
	}
	handler := graphql.NewHandler()
	graphql.RegisterAuthResolvers(handler, authService, fakeUserLookup{})

	got, err := handler.Handle(context.Background(), graphql.ResolverEvent{
		Arguments: map[string]json.RawMessage{
			"input": json.RawMessage(`{
				"email":"user@example.com",
				"password":"password-123",
				"displayName":"Ada",
				"deviceId":"device-1",
				"consents":{"analytics":true}
			}`),
		},
		Info: graphql.ResolverInfo{
			ParentTypeName: "Mutation",
			FieldName:      "register",
		},
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	response, ok := got.(auth.AuthResponse)
	if !ok {
		t.Fatalf("payload = %T, want auth.AuthResponse", got)
	}
	if response.Token != "token" {
		t.Fatalf("Token = %q, want token", response.Token)
	}
	if authService.registerRequest.Email != "user@example.com" ||
		authService.registerRequest.Password != "password-123" ||
		authService.registerRequest.DisplayName != "Ada" ||
		authService.registerRequest.DeviceID != "device-1" ||
		!authService.registerRequest.Consents["analytics"] {
		t.Fatalf("register request mismatch: %+v", authService.registerRequest)
	}
}

func TestRegisterAuthResolversMeChecksTokenVersionAndAccountStatus(t *testing.T) {
	handler := graphql.NewHandler()
	graphql.RegisterAuthResolvers(handler, &fakeAuthService{}, fakeUserLookup{
		user: user.User{
			ID:            "user-1",
			Email:         "user@example.com",
			DisplayName:   "Ada",
			City:          "Izmir",
			TokenVersion:  2,
			AccountStatus: user.AccountStatusActive,
		},
	},
		graphql.WithViewerStats(fakeMeStatsReader{stats: stats.MeStats{
			PointsAvailable: 40,
			RequestsOpen:    2,
			ItemsCompleted:  3,
			City:            "Izmir",
		}}),
		graphql.WithViewerPoints(fakePointsReader{
			balance: points.Balance{UserID: "user-1", Available: 40},
			ledger: []points.Transaction{{
				ID:     "tx-1",
				UserID: "user-1",
				Delta:  40,
				Reason: points.ReasonSignupBonus,
			}},
			nextCursor: "next",
		}),
		graphql.WithViewerSubscription(fakeSubscriptionReader{status: subscription.Status{
			IsActive:  true,
			ProductID: "premium",
		}}, "Sandbox"),
		graphql.WithViewerConsents(fakeConsentReader{state: analytics.ConsentStateResponse{
			Consents: map[analytics.Purpose]analytics.ConsentStatus{
				analytics.PurposeAnalytics: {Granted: true, Version: 1},
			},
		}}),
	)

	got, err := handler.Handle(context.Background(), graphql.ResolverEvent{
		Identity: &graphql.AppSyncIdentity{
			Claims: map[string]any{
				"sub":   "user-1",
				"email": "user@example.com",
				"tv":    2,
			},
		},
		Info: graphql.ResolverInfo{
			ParentTypeName: "Query",
			FieldName:      "me",
		},
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var response struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
		Stats       struct {
			PointsAvailable     int    `json:"pointsAvailable"`
			ActiveRequests      int    `json:"activeRequests"`
			CompletedDeliveries int    `json:"completedDeliveries"`
			City                string `json:"city"`
		} `json:"stats"`
		Points struct {
			Balance struct {
				Available int `json:"available"`
			} `json:"balance"`
			Ledger []struct {
				ID string `json:"id"`
			} `json:"ledger"`
			NextCursor string `json:"nextCursor"`
		} `json:"points"`
		Subscription struct {
			Active      bool   `json:"active"`
			ProductID   string `json:"productId"`
			Environment string `json:"environment"`
		} `json:"subscription"`
		Consents struct {
			Consents map[string]analytics.ConsentStatus `json:"consents"`
		} `json:"consents"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if response.ID != "user-1" || response.Email != "user@example.com" || response.DisplayName != "Ada" ||
		response.Stats.PointsAvailable != 40 || response.Stats.ActiveRequests != 2 ||
		response.Stats.CompletedDeliveries != 3 || response.Stats.City != "Izmir" ||
		response.Points.Balance.Available != 40 || len(response.Points.Ledger) != 1 ||
		response.Points.NextCursor != "next" || !response.Subscription.Active ||
		response.Subscription.ProductID != "premium" || response.Subscription.Environment != "Sandbox" ||
		!response.Consents.Consents["analytics"].Granted {
		t.Fatalf("response mismatch: %+v", response)
	}
}

func TestRegisterAuthResolversLogoutUsesProtectedIdentity(t *testing.T) {
	authService := &fakeAuthService{}
	handler := graphql.NewHandler()
	graphql.RegisterAuthResolvers(handler, authService, fakeUserLookup{
		user: user.User{
			ID:            "user-1",
			Email:         "user@example.com",
			TokenVersion:  2,
			AccountStatus: user.AccountStatusActive,
		},
	})

	got, err := handler.Handle(context.Background(), graphql.ResolverEvent{
		Identity: &graphql.AppSyncIdentity{
			Claims: map[string]any{
				"sub":   "user-1",
				"email": "user@example.com",
				"tv":    2,
			},
		},
		Arguments: map[string]json.RawMessage{
			"input": json.RawMessage(`{"refreshToken":"user-1.refresh-token"}`),
		},
		Info: graphql.ResolverInfo{
			ParentTypeName: "Mutation",
			FieldName:      "logout",
		},
	})
	if err != nil {
		t.Fatalf("logout returned error: %v", err)
	}
	if got != true || authService.logoutRequest.RefreshToken != "user-1.refresh-token" {
		t.Fatalf("logout mismatch: got=%v req=%+v", got, authService.logoutRequest)
	}
}

func TestRegisterAuthResolversPhoneMutationsUseProtectedIdentity(t *testing.T) {
	expiresAt := time.Date(2026, 5, 7, 20, 0, 0, 0, time.UTC)
	authService := &fakeAuthService{
		phoneVerificationResponse: auth.StartPhoneVerificationResponse{
			ExpiresAt:        expiresAt,
			VerificationCode: "123456",
		},
		verifyPhoneChangeUserOutput: user.Response{
			ID:            "user-1",
			Phone:         "+15557770000",
			PhoneVerified: true,
		},
	}
	handler := graphql.NewHandler()
	graphql.RegisterAuthResolvers(handler, authService, fakeUserLookup{
		user: user.User{
			ID:            "user-1",
			Email:         "user@example.com",
			TokenVersion:  2,
			AccountStatus: user.AccountStatusActive,
		},
	})
	identity := &graphql.AppSyncIdentity{
		Claims: map[string]any{
			"sub":   "user-1",
			"email": "user@example.com",
			"tv":    2,
		},
	}

	got, err := handler.Handle(context.Background(), graphql.ResolverEvent{
		Identity: identity,
		Info: graphql.ResolverInfo{
			ParentTypeName: "Mutation",
			FieldName:      "startPhoneVerification",
		},
	})
	if err != nil {
		t.Fatalf("startPhoneVerification returned error: %v", err)
	}
	startPayload, ok := got.(auth.StartPhoneVerificationResponse)
	if !ok {
		t.Fatalf("start payload = %T, want auth.StartPhoneVerificationResponse", got)
	}
	if authService.startPhoneVerificationUser != "user-1" || startPayload.VerificationCode != "123456" {
		t.Fatalf("start phone mismatch: user=%q payload=%+v", authService.startPhoneVerificationUser, startPayload)
	}

	got, err = handler.Handle(context.Background(), graphql.ResolverEvent{
		Identity: identity,
		Arguments: map[string]json.RawMessage{
			"input": json.RawMessage(`{"code":"123456"}`),
		},
		Info: graphql.ResolverInfo{
			ParentTypeName: "Mutation",
			FieldName:      "verifyPhone",
		},
	})
	if err != nil {
		t.Fatalf("verifyPhone returned error: %v", err)
	}
	if got != true || authService.verifyPhoneUser != "user-1" || authService.verifyPhoneRequest.Code != "123456" {
		t.Fatalf("verify phone mismatch: got=%v user=%q req=%+v", got, authService.verifyPhoneUser, authService.verifyPhoneRequest)
	}

	got, err = handler.Handle(context.Background(), graphql.ResolverEvent{
		Identity: identity,
		Arguments: map[string]json.RawMessage{
			"input": json.RawMessage(`{"newPhone":"+1 555 777 0000","password":"step-up"}`),
		},
		Info: graphql.ResolverInfo{
			ParentTypeName: "Mutation",
			FieldName:      "startPhoneChange",
		},
	})
	if err != nil {
		t.Fatalf("startPhoneChange returned error: %v", err)
	}
	if _, ok := got.(auth.StartPhoneVerificationResponse); !ok ||
		authService.startPhoneChangeUser != "user-1" ||
		authService.startPhoneChangeRequest.NewPhone != "+1 555 777 0000" ||
		authService.startPhoneChangeRequest.Password != "step-up" {
		t.Fatalf("start phone change mismatch: got=%T user=%q req=%+v", got, authService.startPhoneChangeUser, authService.startPhoneChangeRequest)
	}

	got, err = handler.Handle(context.Background(), graphql.ResolverEvent{
		Identity: identity,
		Arguments: map[string]json.RawMessage{
			"input": json.RawMessage(`{"code":"654321"}`),
		},
		Info: graphql.ResolverInfo{
			ParentTypeName: "Mutation",
			FieldName:      "verifyPhoneChange",
		},
	})
	if err != nil {
		t.Fatalf("verifyPhoneChange returned error: %v", err)
	}
	changed, ok := got.(user.Response)
	if !ok {
		t.Fatalf("changed payload = %T, want user.Response", got)
	}
	if authService.verifyPhoneChangeUser != "user-1" ||
		authService.verifyPhoneChangeRequest.Code != "654321" ||
		changed.Phone != "+15557770000" ||
		!changed.PhoneVerified {
		t.Fatalf("verify phone change mismatch: user=%q req=%+v payload=%+v", authService.verifyPhoneChangeUser, authService.verifyPhoneChangeRequest, changed)
	}
}

func TestRegisterAuthResolversMeRejectsStaleTokenVersion(t *testing.T) {
	handler := graphql.NewHandler()
	graphql.RegisterAuthResolvers(handler, &fakeAuthService{}, fakeUserLookup{
		user: user.User{
			ID:            "user-1",
			Email:         "user@example.com",
			TokenVersion:  3,
			AccountStatus: user.AccountStatusActive,
		},
	})

	_, err := handler.Handle(context.Background(), graphql.ResolverEvent{
		Identity: &graphql.AppSyncIdentity{
			Claims: map[string]any{
				"sub":   "user-1",
				"email": "user@example.com",
				"tv":    2,
			},
		},
		Info: graphql.ResolverInfo{
			ParentTypeName: "Query",
			FieldName:      "me",
		},
	})

	var appSyncErr *graphql.AppSyncError
	if !errors.As(err, &appSyncErr) {
		t.Fatalf("error = %v, want AppSyncError", err)
	}
	if appSyncErr.ErrorInfo == nil ||
		appSyncErr.ErrorInfo.Status != http.StatusUnauthorized ||
		appSyncErr.ErrorInfo.Code != "unauthorized" {
		t.Fatalf("AppSyncError = %+v, want unauthorized", appSyncErr)
	}
}

func TestRegisterAuthResolversMeMapsMissingUserToUnauthorized(t *testing.T) {
	handler := graphql.NewHandler()
	graphql.RegisterAuthResolvers(handler, &fakeAuthService{}, fakeUserLookup{err: user.ErrNotFound})

	_, err := handler.Handle(context.Background(), graphql.ResolverEvent{
		Identity: &graphql.AppSyncIdentity{
			Claims: map[string]any{
				"sub":   "user-1",
				"email": "user@example.com",
				"tv":    1,
			},
		},
		Info: graphql.ResolverInfo{
			ParentTypeName: "Query",
			FieldName:      "me",
		},
	})

	var appSyncErr *graphql.AppSyncError
	if !errors.As(err, &appSyncErr) {
		t.Fatalf("error = %v, want AppSyncError", err)
	}
	if appSyncErr.ErrorInfo == nil || appSyncErr.ErrorInfo.Status != http.StatusUnauthorized {
		t.Fatalf("AppSyncError = %+v, want unauthorized", appSyncErr)
	}
}

func TestRegisterAuthResolversMeMapsRepositoryErrorToInternal(t *testing.T) {
	handler := graphql.NewHandler()
	graphql.RegisterAuthResolvers(handler, &fakeAuthService{}, fakeUserLookup{err: errors.New("database unavailable")})

	_, err := handler.Handle(context.Background(), graphql.ResolverEvent{
		Identity: &graphql.AppSyncIdentity{
			Claims: map[string]any{
				"sub":   "user-1",
				"email": "user@example.com",
				"tv":    1,
			},
		},
		Info: graphql.ResolverInfo{
			ParentTypeName: "Query",
			FieldName:      "me",
		},
	})

	var appSyncErr *graphql.AppSyncError
	if !errors.As(err, &appSyncErr) {
		t.Fatalf("error = %v, want AppSyncError", err)
	}
	if appSyncErr.ErrorInfo == nil ||
		appSyncErr.ErrorInfo.Status != http.StatusInternalServerError ||
		appSyncErr.ErrorInfo.Code != "internal_error" {
		t.Fatalf("AppSyncError = %+v, want internal", appSyncErr)
	}
}

func TestRegisterAuthResolversPublicMutationRejectsNonPublicOperation(t *testing.T) {
	resolver := graphql.PublicAPIKeyResolver(func(context.Context, graphql.ResolverEvent) (any, error) {
		t.Fatal("resolver should not be called")
		return nil, nil
	})

	_, err := resolver(context.Background(), graphql.ResolverEvent{
		Info: graphql.ResolverInfo{
			ParentTypeName: "Mutation",
			FieldName:      "changePassword",
		},
	})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want AppError", err)
	}
	if appErr.Status != http.StatusForbidden || appErr.Code != "forbidden" {
		t.Fatalf("AppError = %+v, want forbidden", appErr)
	}
}
