package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kindred_server/internal/auth"
)

func TestLocalHTTPHandlerMapsVariableAndInlineArguments(t *testing.T) {
	handler := NewHandler()
	var captured ResolverEvent
	handler.Register("Query", "echo", func(_ context.Context, event ResolverEvent) (any, error) {
		captured = event
		return map[string]any{"ok": true}, nil
	})

	response := executeLocalGraphQL(t, handler, nil, localGraphQLRequest{
		Query: `query Echo($term: String!, $limit: Int!) {
			echo(input: { term: $term, limit: $limit, active: true, tags: ["a", "b"] }, id: "item-1") {
				ok
			}
		}`,
		Variables: map[string]any{
			"term":  "bike",
			"limit": 5,
		},
	}, nil)

	if len(response.Errors) != 0 {
		t.Fatalf("errors = %+v, want none", response.Errors)
	}
	if _, ok := response.Data["echo"]; !ok {
		t.Fatalf("data = %+v, want echo payload", response.Data)
	}
	if captured.Info.ParentTypeName != "Query" || captured.Info.FieldName != "echo" {
		t.Fatalf("operation = %s.%s, want Query.echo", captured.Info.ParentTypeName, captured.Info.FieldName)
	}
	var input map[string]any
	if err := json.Unmarshal(captured.Arguments["input"], &input); err != nil {
		t.Fatalf("decode input argument: %v", err)
	}
	if input["term"] != "bike" || input["limit"] != float64(5) || input["active"] != true {
		t.Fatalf("input = %+v, want evaluated variable and inline values", input)
	}
	var id string
	if err := json.Unmarshal(captured.Arguments["id"], &id); err != nil {
		t.Fatalf("decode id argument: %v", err)
	}
	if id != "item-1" {
		t.Fatalf("id = %q, want item-1", id)
	}
}

func TestLocalHTTPHandlerSelectsNamedOperation(t *testing.T) {
	handler := NewHandler()
	var captured ResolverEvent
	handler.Register("Query", "selected", func(_ context.Context, event ResolverEvent) (any, error) {
		captured = event
		return map[string]any{"id": "selected"}, nil
	})

	response := executeLocalGraphQL(t, handler, nil, localGraphQLRequest{
		Query: `query First {
			first { id }
		}
		query Second($id: ID!) {
			selected(id: $id) { id }
		}`,
		OperationName: "Second",
		Variables:     map[string]any{"id": "chosen"},
	}, nil)

	if len(response.Errors) != 0 {
		t.Fatalf("errors = %+v, want none", response.Errors)
	}
	var id string
	if err := json.Unmarshal(captured.Arguments["id"], &id); err != nil {
		t.Fatalf("decode id argument: %v", err)
	}
	if id != "chosen" {
		t.Fatalf("id = %q, want chosen", id)
	}
}

func TestLocalHTTPHandlerAllowsPublicMutationWithoutBearer(t *testing.T) {
	handler := NewHandler()
	handler.Register("Mutation", "login", PublicAPIKeyResolver(func(_ context.Context, _ ResolverEvent) (any, error) {
		return map[string]any{"token": "local-token"}, nil
	}))
	verifier := &fakeTokenVerifier{}

	response := executeLocalGraphQL(t, handler, verifier, localGraphQLRequest{
		Query: `mutation Login($input: LoginInput!) {
			login(input: $input) { token }
		}`,
		Variables: map[string]any{"input": map[string]any{
			"email":    "dev@example.com",
			"password": "password",
		}},
	}, nil)

	if len(response.Errors) != 0 {
		t.Fatalf("errors = %+v, want none", response.Errors)
	}
	if len(verifier.tokens) != 0 {
		t.Fatalf("VerifyToken calls = %v, want none", verifier.tokens)
	}
	if _, ok := response.Data["login"]; !ok {
		t.Fatalf("data = %+v, want login payload", response.Data)
	}
}

func TestLocalHTTPHandlerPopulatesBearerIdentity(t *testing.T) {
	handler := NewHandler()
	var capturedClaims auth.Claims
	var capturedIdentity *AppSyncIdentity
	handler.Register("Query", "me", ProtectedResolver(func(_ context.Context, event ResolverEvent, claims auth.Claims) (any, error) {
		capturedClaims = claims
		capturedIdentity = event.Identity
		return map[string]any{"id": claims.UserID}, nil
	}))
	verifier := &fakeTokenVerifier{claims: auth.Claims{
		UserID:       "user-1",
		Email:        "user@example.com",
		TokenVersion: 3,
		DeviceID:     "device-1",
	}}

	response := executeLocalGraphQL(t, handler, verifier, localGraphQLRequest{
		Query: `query Me { me { id } }`,
	}, map[string]string{
		"Authorization":   "Bearer local-token",
		"X-Forwarded-For": "203.0.113.7",
	})

	if len(response.Errors) != 0 {
		t.Fatalf("errors = %+v, want none", response.Errors)
	}
	if len(verifier.tokens) != 1 || verifier.tokens[0] != "local-token" {
		t.Fatalf("VerifyToken calls = %v, want local-token", verifier.tokens)
	}
	if capturedClaims.UserID != "user-1" || capturedClaims.Email != "user@example.com" || capturedClaims.TokenVersion != 3 || capturedClaims.DeviceID != "device-1" {
		t.Fatalf("claims = %+v, want verifier claims", capturedClaims)
	}
	if capturedIdentity == nil {
		t.Fatal("identity = nil, want AppSync identity")
	}
	if capturedIdentity.Sub != "user-1" || capturedIdentity.Claims["email"] != "user@example.com" || capturedIdentity.Claims["tv"] != 3 || capturedIdentity.Claims["did"] != "device-1" {
		t.Fatalf("identity = %+v, want mapped claims", capturedIdentity)
	}
	if got := capturedIdentity.SourceIP; len(got) != 1 || got[0] != "203.0.113.7" {
		t.Fatalf("sourceIP = %v, want forwarded IP", got)
	}
}

func TestLocalHTTPHandlerReturnsGraphQLErrorShapes(t *testing.T) {
	tests := []struct {
		name       string
		handler    *Handler
		request    localGraphQLRequest
		wantCode   string
		wantStatus float64
	}{
		{
			name:       "unknown field",
			handler:    NewHandler(),
			request:    localGraphQLRequest{Query: `query Missing { missing { id } }`},
			wantCode:   "unknown_field",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "malformed query",
			handler:    NewHandler(),
			request:    localGraphQLRequest{Query: `query Broken {`},
			wantCode:   "bad_request",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "subscription",
			handler:    NewHandler(),
			request:    localGraphQLRequest{Query: `subscription MessageCreated { messageCreated(requestId: "request-1") { id } }`},
			wantCode:   "unsupported_operation",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "appsync error",
			handler: func() *Handler {
				handler := NewHandler()
				handler.Register("Query", "fail", func(context.Context, ResolverEvent) (any, error) {
					return nil, NewAppSyncError(http.StatusConflict, "conflict", "already exists")
				})
				return handler
			}(),
			request:    localGraphQLRequest{Query: `query Fail { fail { id } }`},
			wantCode:   "conflict",
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := executeLocalGraphQL(t, tt.handler, nil, tt.request, nil)
			if len(response.Errors) != 1 {
				t.Fatalf("errors = %+v, want one error", response.Errors)
			}
			got := response.Errors[0].Extensions
			if got["code"] != tt.wantCode || got["status"] != tt.wantStatus {
				t.Fatalf("extensions = %+v, want code %q status %.0f", got, tt.wantCode, tt.wantStatus)
			}
		})
	}
}

type localHTTPTestResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []struct {
		Message    string         `json:"message"`
		Extensions map[string]any `json:"extensions"`
	} `json:"errors"`
}

type fakeTokenVerifier struct {
	claims auth.Claims
	err    error
	tokens []string
}

func (f *fakeTokenVerifier) VerifyToken(_ context.Context, token string) (auth.Claims, error) {
	f.tokens = append(f.tokens, token)
	return f.claims, f.err
}

func executeLocalGraphQL(
	t *testing.T,
	handler *Handler,
	verifier TokenVerifier,
	request localGraphQLRequest,
	headers map[string]string,
) localHTTPTestResponse {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()

	NewLocalHTTPHandler(handler, verifier).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", recorder.Code, recorder.Body.String())
	}
	var response localHTTPTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}
