package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestDecodedCanonicalPathParamRestoresServerOwnedIdentity(t *testing.T) {
	route := chi.NewRouteContext()
	route.URLParams.Add("id", "package%3Aassignment%3Aplan-intake-001")
	request := (&http.Request{}).WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, route))

	if got, want := decodedCanonicalPathParam(request, "id"), "package:assignment:plan-intake-001"; got != want {
		t.Fatalf("decoded canonical path identity = %q, want %q", got, want)
	}
}
