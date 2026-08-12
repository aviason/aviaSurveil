//go:build canonicaltest

package httpapi

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/aviason/aviaSurveil/internal/testprofile"
)

const (
	CanonicalTestSubjectHeader = "X-Avia-Test-Subject"
	CanonicalTestTokenHeader   = "X-Avia-Test-Token"
)

type CanonicalTestBoundary struct {
	token string
}

func NewCanonicalTestBoundary(token string) *CanonicalTestBoundary {
	return &CanonicalTestBoundary{token: token}
}

func (boundary *CanonicalTestBoundary) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !boundary.validToken(request.Header.Get(CanonicalTestTokenHeader)) {
			writeProblem(writer, http.StatusUnauthorized, "Authentication required", "canonical test token is missing or invalid", "UNAUTHENTICATED")
			return
		}
		principal, ok := testprofile.Principal(strings.TrimSpace(request.Header.Get(CanonicalTestSubjectHeader)))
		if !ok {
			writeProblem(writer, http.StatusForbidden, "Request forbidden", "test subject is not in the server-owned canonical profile", "TEST_SUBJECT_FORBIDDEN")
			return
		}
		ctx := context.WithValue(request.Context(), principalContextKey{}, principal)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func (boundary *CanonicalTestBoundary) Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !boundary.validToken(request.Header.Get(CanonicalTestTokenHeader)) {
			writeProblem(writer, http.StatusUnauthorized, "Authentication required", "canonical test token is missing or invalid", "UNAUTHENTICATED")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (boundary *CanonicalTestBoundary) validToken(candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	return boundary.token != "" && subtle.ConstantTimeCompare([]byte(candidate), []byte(boundary.token)) == 1
}
