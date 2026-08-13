package identity

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/password"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgreSQLStoreLifecycle(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AVIA_AUTH_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("not run: AVIA_AUTH_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open disposable PostgreSQL: %v", err)
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply identity migration: %v", err)
	}
	hasher, err := password.New(password.Params{MemoryKiB: 16 * 1024, Time: 1, Threads: 1, KeyLength: 32, SaltLen: 16, MaxBytes: 1024, Capacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	limiter, err := throttle.NewMemoryLimiter(time.Minute, 100, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewPostgresStore(pool, Config{
		Hasher:         hasher,
		PasswordPolicy: password.DefaultPolicy(),
		Limiter:        limiter,
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("198.51.100.0/24")},
	})
	if err != nil {
		t.Fatal(err)
	}
	email := "pg-" + strings.ToLower(strings.ReplaceAll(time.Now().UTC().Format("150405.000000"), ".", "")) + "@example.invalid"
	account, _, err := store.ProvisionInvitation(ctx, InvitationInput{Email: email, Username: "pg" + strings.TrimSuffix(strings.TrimPrefix(email, "pg-"), "@example.invalid")})
	if err != nil {
		t.Fatal(err)
	}
	account, err = store.SetEmailVerified(ctx, account.SubjectID, account.AuthRevision)
	if err != nil {
		t.Fatal(err)
	}
	account, err = store.Activate(ctx, account.SubjectID, account.AuthRevision, []byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Authenticate(ctx, AuthenticationRequest{Identifier: email, Password: []byte("correct horse battery staple"), Source: throttle.ForwardedHeaders{RemoteAddr: "203.0.113.9:443"}, DeviceKey: "pg-device"})
	if err != nil || result.Account.SubjectID != account.SubjectID {
		t.Fatalf("PostgreSQL authentication = %+v/%v", result, err)
	}
	changed, err := store.ChangePassword(ctx, account.SubjectID, account.AuthRevision, []byte("correct horse battery staple"), []byte("new correct password 2"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ChangePassword(ctx, account.SubjectID, account.AuthRevision, []byte("new correct password 2"), []byte("another correct password 3")); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale PostgreSQL password change = %v", err)
	}
	if _, err := store.Authenticate(ctx, AuthenticationRequest{Identifier: email, Password: []byte("new correct password 2"), Source: throttle.ForwardedHeaders{RemoteAddr: "203.0.113.9:443"}, DeviceKey: "pg-device"}); err != nil {
		t.Fatalf("authentication after password change = %v", err)
	}
	reset, err := store.ResetPassword(ctx, account.SubjectID, changed.AuthRevision, []byte("reset correct password 3"))
	if err != nil {
		t.Fatalf("PostgreSQL password reset = %v", err)
	}
	if reset.AuthRevision <= changed.AuthRevision {
		t.Fatalf("password reset revision = %d, want advance from %d", reset.AuthRevision, changed.AuthRevision)
	}
	if _, err := store.Authenticate(ctx, AuthenticationRequest{Identifier: email, Password: []byte("reset correct password 3"), Source: throttle.ForwardedHeaders{RemoteAddr: "203.0.113.9:443"}, DeviceKey: "pg-device"}); err != nil {
		t.Fatalf("authentication after password reset = %v", err)
	}
	if _, err := store.Transition(ctx, account.SubjectID, reset.AuthRevision, AccountSuspended); err != nil {
		t.Fatal(err)
	}
}
