package qualification

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/challenge"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mfa"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/password"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestRuntimeBrowserSeed is invoked only by the disposable browser runner from
// inside the candidate's private Docker network. It creates no repository or
// normal-profile state and writes the current synthetic MFA code only to the
// runner-owned file supplied by the test harness.
func TestRuntimeBrowserSeed(t *testing.T) {
	if os.Getenv("AVIA_AUTH_RUNTIME_BROWSER_SEED") != "1" {
		t.Skip("not run: browser seed was not requested")
	}
	databasePath := os.Getenv("AVIA_AUTH_TEST_DATABASE_URL_FILE")
	passwordPath := os.Getenv("AVIA_AUTH_RUNTIME_BROWSER_PASSWORD_FILE")
	mfaKeyPath := os.Getenv("AVIA_AUTH_RUNTIME_MFA_KEY_FILE")
	codePath := os.Getenv("AVIA_AUTH_RUNTIME_MFA_CODE_FILE")
	passwordResetPath := os.Getenv("AVIA_AUTH_RUNTIME_PASSWORD_RESET_URL_FILE")
	mfaResetPath := os.Getenv("AVIA_AUTH_RUNTIME_MFA_RESET_URL_FILE")
	issuer := strings.TrimRight(os.Getenv("AVIA_AUTH_RUNTIME_BROWSER_ISSUER"), "/")
	if databasePath == "" || passwordPath == "" || mfaKeyPath == "" || codePath == "" || passwordResetPath == "" || mfaResetPath == "" || issuer == "" {
		t.Fatal("browser seed file paths are required")
	}
	databaseURL, err := os.ReadFile(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	accountPassword, err := os.ReadFile(passwordPath)
	if err != nil {
		t.Fatal(err)
	}
	mfaKey, err := os.ReadFile(mfaKeyPath)
	if err != nil || len(mfaKey) != 32 {
		t.Fatal("browser seed MFA key is unavailable")
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
	account, _, err := identities.ProvisionInvitation(ctx, identity.InvitationInput{Email: "browser-candidate@example.invalid", Username: "browsercandidate"})
	if err != nil {
		t.Fatal(err)
	}
	account, err = identities.SetEmailVerified(ctx, account.SubjectID, account.AuthRevision)
	if err != nil {
		t.Fatal(err)
	}
	account, err = identities.Activate(ctx, account.SubjectID, account.AuthRevision, []byte(strings.TrimSpace(string(accountPassword))))
	if err != nil {
		t.Fatal(err)
	}
	seedNow := time.Now().Add(-30 * time.Second)
	factors, err := mfa.NewPostgresStore(pool, mfa.Config{EncryptionKey: mfaKey, Clock: func() time.Time { return seedNow }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := factors.Enroll(ctx, account.SubjectID, "AviaSurveil360", "browser-candidate@example.invalid"); err != nil {
		t.Fatal(err)
	}
	confirmation, err := factors.CurrentCodeForTesting(ctx, account.SubjectID, seedNow)
	if err != nil {
		t.Fatal(err)
	}
	if err := factors.ConfirmEnrollment(ctx, account.SubjectID, confirmation); err != nil {
		t.Fatal(err)
	}
	current, err := factors.CurrentCodeForTesting(ctx, account.SubjectID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codePath, []byte(current+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	challenges, err := challenge.NewPostgresStore(pool, challenge.Config{})
	if err != nil {
		t.Fatal(err)
	}
	passwordReset, err := challenges.Issue(ctx, account.SubjectID, challenge.PurposePasswordReset, 10*time.Minute, 3)
	if err != nil {
		t.Fatal(err)
	}
	mfaReset, err := challenges.Issue(ctx, account.SubjectID, challenge.PurposeMFARecovery, 10*time.Minute, 3)
	if err != nil {
		t.Fatal(err)
	}
	passwordResetURL := issuer + "/recover/password?" + url.Values{"subject": {account.SubjectID}, "token": {passwordReset.Token}}.Encode()
	mfaResetURL := issuer + "/recover/mfa?" + url.Values{"subject": {account.SubjectID}, "token": {mfaReset.Token}}.Encode()
	if err := os.WriteFile(passwordResetPath, []byte(passwordResetURL+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mfaResetPath, []byte(mfaResetURL+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
