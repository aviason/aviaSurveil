package graphql_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"kindred_server/internal/auth"
	"kindred_server/internal/graphql"
	apperrors "kindred_server/internal/platform/errors"
)

type registerInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func TestDecodeInputDecodesAppSyncArgumentsShape(t *testing.T) {
	var event graphql.ResolverEvent
	raw := []byte(`{
		"arguments": {
			"input": {
				"email": "user@example.com",
				"password": "fixture-secret"
			}
		}
	}`)
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	input, err := graphql.DecodeInput[registerInput](event)
	if err != nil {
		t.Fatalf("DecodeInput returned error: %v", err)
	}
	if input.Email != "user@example.com" || input.Password != "fixture-secret" {
		t.Fatalf("input = %+v, want decoded input", input)
	}
}

func TestDecodeArgumentDecodesRawMessageArgument(t *testing.T) {
	event := graphql.ResolverEvent{
		Arguments: map[string]json.RawMessage{
			"limit": json.RawMessage(`25`),
		},
	}

	limit, err := graphql.DecodeArgument[int](event, "limit")
	if err != nil {
		t.Fatalf("DecodeArgument returned error: %v", err)
	}
	if limit != 25 {
		t.Fatalf("limit = %d, want 25", limit)
	}
}

func TestDecodeArgumentReturnsBadRequestForMissingOrInvalidArgument(t *testing.T) {
	tests := []struct {
		name  string
		event graphql.ResolverEvent
	}{
		{
			name: "missing argument",
		},
		{
			name: "null argument",
			event: graphql.ResolverEvent{Arguments: map[string]json.RawMessage{
				"input": json.RawMessage(`null`),
			}},
		},
		{
			name: "invalid json",
			event: graphql.ResolverEvent{Arguments: map[string]json.RawMessage{
				"input": json.RawMessage(`{"email":`),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := graphql.DecodeInput[registerInput](tt.event)
			var appErr *apperrors.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("error = %v, want AppError", err)
			}
			if appErr.Status != http.StatusBadRequest || appErr.Code != "bad_request" {
				t.Fatalf("AppError = %+v, want bad_request", appErr)
			}
		})
	}
}

func TestRequireOIDCIdentityParsesClaims(t *testing.T) {
	var event graphql.ResolverEvent
	raw := []byte(`{
		"identity": {
			"claims": {
				"sub": "user-1",
				"email": "user@example.com",
				"did": "device-1",
				"tv": 7
			}
		}
	}`)
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	claims, err := graphql.RequireOIDCIdentity(context.Background(), event)
	if err != nil {
		t.Fatalf("RequireOIDCIdentity returned error: %v", err)
	}
	if claims.UserID != "user-1" ||
		claims.Email != "user@example.com" ||
		claims.DeviceID != "device-1" ||
		claims.TokenVersion != 7 {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestRequireOIDCIdentityUsesTopLevelSubAndStringTokenVersion(t *testing.T) {
	event := graphql.ResolverEvent{
		Identity: &graphql.AppSyncIdentity{
			Sub: "user-2",
			Claims: map[string]any{
				"email": "second@example.com",
				"tv":    "12",
			},
		},
	}

	claims, err := graphql.RequireOIDCIdentity(context.Background(), event)
	if err != nil {
		t.Fatalf("RequireOIDCIdentity returned error: %v", err)
	}
	if claims.UserID != "user-2" || claims.Email != "second@example.com" ||
		claims.TokenVersion != 12 || claims.DeviceID != "" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestRequireOIDCIdentityReturnsUnauthorizedAppError(t *testing.T) {
	tests := []struct {
		name  string
		event graphql.ResolverEvent
	}{
		{
			name: "missing identity",
		},
		{
			name: "missing sub",
			event: graphql.ResolverEvent{Identity: &graphql.AppSyncIdentity{
				Claims: map[string]any{"email": "user@example.com", "tv": 1},
			}},
		},
		{
			name: "invalid token version",
			event: graphql.ResolverEvent{Identity: &graphql.AppSyncIdentity{
				Claims: map[string]any{"sub": "user-1", "email": "user@example.com", "tv": "nope"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := graphql.RequireOIDCIdentity(context.Background(), tt.event)
			var appErr *apperrors.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("error = %v, want AppError", err)
			}
			if appErr.Status != http.StatusUnauthorized || appErr.Code != "unauthorized" {
				t.Fatalf("AppError = %+v, want unauthorized", appErr)
			}
		})
	}
}

func TestProtectedResolverPassesParsedClaims(t *testing.T) {
	event := graphql.ResolverEvent{
		Identity: &graphql.AppSyncIdentity{
			Claims: map[string]any{
				"sub":   "user-1",
				"email": "user@example.com",
				"tv":    3,
			},
		},
	}
	resolver := graphql.ProtectedResolver(func(ctx context.Context, gotEvent graphql.ResolverEvent, claims auth.Claims) (any, error) {
		if gotEvent.Identity == nil {
			t.Fatalf("event was not passed through")
		}
		return claims.UserID + ":" + claims.Email, nil
	})

	got, err := resolver(context.Background(), event)
	if err != nil {
		t.Fatalf("resolver returned error: %v", err)
	}
	if got != "user-1:user@example.com" {
		t.Fatalf("payload = %v, want parsed claims payload", got)
	}
}

func TestProtectedResolverRejectsMissingIdentity(t *testing.T) {
	resolver := graphql.ProtectedResolver(func(context.Context, graphql.ResolverEvent, auth.Claims) (any, error) {
		t.Fatal("protected resolver should not be called")
		return nil, nil
	})

	_, err := resolver(context.Background(), graphql.ResolverEvent{})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want AppError", err)
	}
	if appErr.Status != http.StatusUnauthorized || appErr.Code != "unauthorized" {
		t.Fatalf("AppError = %+v, want unauthorized", appErr)
	}
}

func TestIsAPIKeyPublicOperation(t *testing.T) {
	publicMutations := []string{
		"register",
		"login",
		"reactivate",
		"refreshSession",
		"verifyEmail",
		"resendEmailVerification",
		"forgotPassword",
		"resetPassword",
	}
	for _, fieldName := range publicMutations {
		t.Run(fieldName, func(t *testing.T) {
			event := graphql.ResolverEvent{
				Info: graphql.ResolverInfo{
					ParentTypeName: "Mutation",
					FieldName:      fieldName,
				},
			}
			if !graphql.IsAPIKeyPublicOperation(event) {
				t.Fatalf("IsAPIKeyPublicOperation(%q) = false, want true", fieldName)
			}
		})
	}
	publicQueries := []string{
		"keyTransparencyTreeHead",
		"keyTransparencyLog",
	}
	for _, fieldName := range publicQueries {
		t.Run(fieldName, func(t *testing.T) {
			event := graphql.ResolverEvent{
				Info: graphql.ResolverInfo{
					ParentTypeName: "Query",
					FieldName:      fieldName,
				},
			}
			if !graphql.IsAPIKeyPublicOperation(event) {
				t.Fatalf("IsAPIKeyPublicOperation(%q) = false, want true", fieldName)
			}
		})
	}

	notPublic := []graphql.ResolverEvent{
		{Info: graphql.ResolverInfo{ParentTypeName: "Query", FieldName: "login"}},
		{Info: graphql.ResolverInfo{ParentTypeName: "Mutation", FieldName: "keyTransparencyTreeHead"}},
		{Info: graphql.ResolverInfo{ParentTypeName: "Mutation", FieldName: "changePassword"}},
		{Info: graphql.ResolverInfo{ParentTypeName: "Mutation"}},
	}
	for _, event := range notPublic {
		if graphql.IsAPIKeyPublicOperation(event) {
			t.Fatalf("IsAPIKeyPublicOperation(%+v) = true, want false", event)
		}
	}
}

func TestPublicAPIKeyResolverAllowsOnlyPublicOperations(t *testing.T) {
	resolver := graphql.PublicAPIKeyResolver(func(context.Context, graphql.ResolverEvent) (any, error) {
		return "ok", nil
	})
	for _, event := range []graphql.ResolverEvent{
		{Info: graphql.ResolverInfo{ParentTypeName: "Mutation", FieldName: "login"}},
		{Info: graphql.ResolverInfo{ParentTypeName: "Query", FieldName: "keyTransparencyLog"}},
	} {
		got, err := resolver(context.Background(), event)
		if err != nil {
			t.Fatalf("public resolver returned error for %+v: %v", event, err)
		}
		if got != "ok" {
			t.Fatalf("payload = %v, want ok", got)
		}
	}

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

func TestIsAPIKeyPublicOperationUsesTopLevelFallback(t *testing.T) {
	event := graphql.ResolverEvent{
		TypeName:  "Mutation",
		FieldName: "login",
	}
	if !graphql.IsAPIKeyPublicOperation(event) {
		t.Fatal("IsAPIKeyPublicOperation with top-level fields = false, want true")
	}
}

func TestAPIKeyPublicOperationsMatchSchemaDirectives(t *testing.T) {
	raw, err := os.ReadFile("../../graphql/schema.graphqls")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	re := regexp.MustCompile(`(?m)^\s*([A-Za-z][A-Za-z0-9_]*)\s*(?:\([^)]*\))?\s*:.*@aws_api_key`)
	matches := re.FindAllStringSubmatch(string(raw), -1)
	if len(matches) != 10 {
		t.Fatalf("schema public operation count = %d, want 10", len(matches))
	}
	for _, match := range matches {
		fieldName := match[1]
		parentType := "Mutation"
		if strings.HasPrefix(fieldName, "keyTransparency") {
			parentType = "Query"
		}
		event := graphql.ResolverEvent{
			Info: graphql.ResolverInfo{
				ParentTypeName: parentType,
				FieldName:      fieldName,
			},
		}
		if !graphql.IsAPIKeyPublicOperation(event) {
			t.Fatalf("schema marks %q with @aws_api_key but helper returns false", fieldName)
		}
	}
}
