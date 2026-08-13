package qualification

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/password"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestRuntimeLoadSeed creates one factor-free account for the bounded,
// disposable ARM64 auth-runtime load qualification. It is never run by the
// ordinary package suite and writes no state outside the supplied test runtime.
func TestRuntimeLoadSeed(t *testing.T) {
	if os.Getenv("AVIA_AUTH_RUNTIME_LOAD_SEED") != "1" {
		t.Skip("not run: runtime load seed was not requested")
	}
	databasePath := os.Getenv("AVIA_AUTH_TEST_DATABASE_URL_FILE")
	passwordPath := os.Getenv("AVIA_AUTH_RUNTIME_LOAD_PASSWORD_FILE")
	if databasePath == "" || passwordPath == "" {
		t.Fatal("runtime load seed file paths are required")
	}
	databaseURL, err := os.ReadFile(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	accountPassword, err := os.ReadFile(passwordPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, strings.TrimSpace(string(databaseURL)))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	hasher, err := password.NewDefault()
	if err != nil {
		t.Fatal(err)
	}
	limiter, err := throttle.NewMemoryLimiter(time.Minute, 100, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	identities, err := identity.NewPostgresStore(pool, identity.Config{Hasher: hasher, PasswordPolicy: password.DefaultPolicy(), Limiter: limiter})
	if err != nil {
		t.Fatal(err)
	}
	account, invitation, err := identities.ProvisionProviderInvitation(
		ctx,
		identity.InvitationInput{Email: "load-candidate@example.invalid", Username: "loadcandidate"},
		identity.ProviderProfileInput{DisplayName: "Load Candidate", GivenName: "Load", FamilyName: "Candidate"},
		identity.ProviderAuthorityInput{MembershipID: "membership-load-candidate", OrganizationID: "CAA", Role: "inspector", ExpectedRevision: 0, ResultingRevision: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := identities.ActivateWithInvitation(ctx, account.SubjectID, invitation.Token, []byte(strings.TrimSpace(string(accountPassword)))); err != nil {
		t.Fatal(err)
	}
}
