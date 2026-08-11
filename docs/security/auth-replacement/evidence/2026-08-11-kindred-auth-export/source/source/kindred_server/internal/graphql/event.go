package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"kindred_server/internal/auth"
	apperrors "kindred_server/internal/platform/errors"
)

type ResolverEvent struct {
	Arguments map[string]json.RawMessage `json:"arguments,omitempty"`
	Identity  *AppSyncIdentity           `json:"identity,omitempty"`
	Request   ResolverRequest            `json:"request,omitempty"`
	Info      ResolverInfo               `json:"info,omitempty"`

	TypeName  string `json:"typeName,omitempty"`
	FieldName string `json:"fieldName,omitempty"`
}

type ResolverRequest struct {
	Headers map[string]string `json:"headers,omitempty"`
}

type ResolverInfo struct {
	FieldName           string         `json:"fieldName,omitempty"`
	ParentTypeName      string         `json:"parentTypeName,omitempty"`
	SelectionSetList    []string       `json:"selectionSetList,omitempty"`
	SelectionSetGraphQL string         `json:"selectionSetGraphQL,omitempty"`
	Variables           map[string]any `json:"variables,omitempty"`
	Extensions          map[string]any `json:"extensions,omitempty"`
}

type AppSyncIdentity struct {
	Sub                 string         `json:"sub,omitempty"`
	Issuer              string         `json:"issuer,omitempty"`
	Username            string         `json:"username,omitempty"`
	Claims              map[string]any `json:"claims,omitempty"`
	SourceIP            []string       `json:"sourceIp,omitempty"`
	DefaultAuthStrategy string         `json:"defaultAuthStrategy,omitempty"`
}

func DecodeArgument[T any](event ResolverEvent, name string) (T, error) {
	var value T
	raw, ok := event.Arguments[name]
	if !ok {
		return value, apperrors.BadRequest("missing GraphQL argument " + name)
	}
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return value, apperrors.BadRequest("missing GraphQL argument " + name)
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, apperrors.BadRequest("invalid GraphQL argument " + name)
	}
	return value, nil
}

func DecodeInput[T any](event ResolverEvent) (T, error) {
	return DecodeArgument[T](event, "input")
}

func isNullRawMessage(raw json.RawMessage) bool {
	return len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null"
}

func RequireOIDCIdentity(_ context.Context, event ResolverEvent) (auth.Claims, error) {
	if event.Identity == nil {
		return auth.Claims{}, apperrors.Unauthorized("invalid or missing identity")
	}
	claims, ok := claimsFromIdentity(event.Identity)
	if !ok {
		return auth.Claims{}, apperrors.Unauthorized("invalid or missing identity")
	}
	return claims, nil
}

type ProtectedResolverFunc func(context.Context, ResolverEvent, auth.Claims) (any, error)

func ProtectedResolver(resolver ProtectedResolverFunc) ResolverFunc {
	return func(ctx context.Context, event ResolverEvent) (any, error) {
		claims, err := RequireOIDCIdentity(ctx, event)
		if err != nil {
			return nil, err
		}
		return resolver(ctx, event, claims)
	}
}

func PublicAPIKeyResolver(resolver ResolverFunc) ResolverFunc {
	return func(ctx context.Context, event ResolverEvent) (any, error) {
		if err := RequireAPIKeyPublicOperation(ctx, event); err != nil {
			return nil, err
		}
		return resolver(ctx, event)
	}
}

func RequireAPIKeyPublicOperation(_ context.Context, event ResolverEvent) error {
	if !IsAPIKeyPublicOperation(event) {
		return apperrors.Forbidden("GraphQL operation is not public")
	}
	return nil
}

func IsAPIKeyPublicOperation(event ResolverEvent) bool {
	parentType, fieldName := operation(event)
	switch parentType {
	case "Mutation":
		return publicAPIKeyMutations[fieldName]
	case "Query":
		return publicAPIKeyQueries[fieldName]
	default:
		return false
	}
}

var publicAPIKeyQueries = map[string]bool{
	"keyTransparencyTreeHead": true,
	"keyTransparencyLog":      true,
}

var publicAPIKeyMutations = map[string]bool{
	"register":                true,
	"login":                   true,
	"reactivate":              true,
	"refreshSession":          true,
	"verifyEmail":             true,
	"resendEmailVerification": true,
	"forgotPassword":          true,
	"resetPassword":           true,
}

func claimsFromIdentity(identity *AppSyncIdentity) (auth.Claims, bool) {
	sub, ok := claimString(identity.Claims, "sub")
	if !ok {
		sub = strings.TrimSpace(identity.Sub)
		ok = sub != ""
	}
	if !ok {
		return auth.Claims{}, false
	}

	email, ok := claimString(identity.Claims, "email")
	if !ok {
		return auth.Claims{}, false
	}
	tokenVersion, ok := claimInt(identity.Claims, "tv")
	if !ok {
		return auth.Claims{}, false
	}
	deviceID, _ := claimString(identity.Claims, "did")

	return auth.Claims{
		UserID:       sub,
		Email:        email,
		TokenVersion: tokenVersion,
		DeviceID:     deviceID,
	}, true
}

func claimString(claims map[string]any, name string) (string, bool) {
	value, ok := claims[name]
	if !ok {
		return "", false
	}
	s, ok := value.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	return s, s != ""
}

func claimInt(claims map[string]any, name string) (int, bool) {
	value, ok := claims[name]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, v >= 0
	case int64:
		return int(v), v >= 0 && int64(int(v)) == v
	case float64:
		i := int(v)
		return i, v >= 0 && float64(i) == v
	case json.Number:
		i, err := strconv.ParseInt(v.String(), 10, 0)
		if err != nil || i < 0 {
			return 0, false
		}
		return int(i), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || i < 0 {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func operation(event ResolverEvent) (string, string) {
	parentType := event.Info.ParentTypeName
	if parentType == "" {
		parentType = event.TypeName
	}
	fieldName := event.Info.FieldName
	if fieldName == "" {
		fieldName = event.FieldName
	}
	return parentType, fieldName
}
