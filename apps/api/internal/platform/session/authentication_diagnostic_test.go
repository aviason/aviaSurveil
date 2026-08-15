package session

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestAuthenticationFailureDiagnosticIsFixedAndPreservesTheSentinel(t *testing.T) {
	t.Parallel()

	err := authenticationFailure("invalid-role", ErrUnauthenticated)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("wrapped rejection no longer matches ErrUnauthenticated: %v", err)
	}
	if got := AuthenticationFailureDiagnostic(err); got != "invalid-role" {
		t.Fatalf("diagnostic = %q, want invalid-role", got)
	}
	if got := AuthenticationFailureDiagnostic(errors.New("database unavailable")); got != "internal" {
		t.Fatalf("unclassified diagnostic = %q, want internal", got)
	}
	if got := AuthenticationFailureDiagnostic(fmt.Errorf("wrapped: %w", ErrUnauthenticated)); got != "unauthenticated" {
		t.Fatalf("sentinel diagnostic = %q, want unauthenticated", got)
	}
	if got := AuthenticationFailureDiagnostic(&pgconn.PgError{Code: "42501", Message: "permission denied for table session_references"}); got != "postgres-privilege-session-references" {
		t.Fatalf("PostgreSQL diagnostic = %q, want postgres-privilege-session-references", got)
	}
	for _, test := range []struct {
		name     string
		err      error
		expected string
	}{
		{
			name: "department assignment remains the outer stage",
			err: fmt.Errorf(
				"resolve authenticated department authority: %w",
				&pgconn.PgError{Code: "42501"},
			),
			expected: "department-assignment",
		},
		{
			name:     "public schema",
			err:      &pgconn.PgError{Code: "42501", Message: "permission denied for schema public"},
			expected: "postgres-privilege-public-schema",
		},
		{
			name:     "transaction trace context",
			err:      &pgconn.PgError{Code: "42501", Message: "permission denied to set parameter \"avia.traceparent\""},
			expected: "postgres-privilege-trace-context",
		},
		{
			name:     "audit sequence",
			err:      &pgconn.PgError{Code: "42501", Message: "permission denied for sequence audit_events_sequence_id_seq"},
			expected: "postgres-privilege-audit-sequence",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := AuthenticationFailureDiagnostic(test.err); got != test.expected {
				t.Fatalf("diagnostic = %q, want %q", got, test.expected)
			}
		})
	}
	if got := AuthenticationFailureDiagnostic(&pgconn.PgError{Code: "42501"}); got != "postgres-insufficient-privilege" {
		t.Fatalf("generic PostgreSQL diagnostic = %q, want postgres-insufficient-privilege", got)
	}
	if got := AuthenticationFailureDiagnostic(context.DeadlineExceeded); got != "context-expired" {
		t.Fatalf("context diagnostic = %q, want context-expired", got)
	}
	if got := AuthenticationFailureDiagnostic(authenticationFailure("transaction", context.Canceled)); got != "context-expired" {
		t.Fatalf("cancelled transaction diagnostic = %q, want context-expired", got)
	}
	if got := AuthenticationFailureDiagnostic(errors.New("resolve authenticated department authority: failed")); got != "department-assignment" {
		t.Fatalf("department diagnostic = %q, want department-assignment", got)
	}
}
