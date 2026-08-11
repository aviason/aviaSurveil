package identity

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/password"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
)

type testStoreFixture struct {
	store *Store
	now   *time.Time
}

func newTestStore(t *testing.T) testStoreFixture {
	t.Helper()
	hasher, err := password.New(password.Params{MemoryKiB: 16 * 1024, Time: 1, Threads: 1, KeyLength: 32, SaltLen: 16, MaxBytes: 1024, Capacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1700000000, 0)
	limiter, err := throttle.NewMemoryLimiter(time.Minute, 100, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(Config{
		Hasher:         hasher,
		PasswordPolicy: password.DefaultPolicy(),
		Limiter:        limiter,
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("198.51.100.0/24")},
		Clock:          func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return testStoreFixture{store: store, now: &now}
}

func activateTestAccount(t *testing.T, fixture testStoreFixture) AccountSnapshot {
	t.Helper()
	ctx := context.Background()
	account, _, err := fixture.store.ProvisionInvitation(ctx, InvitationInput{Email: "alice@example.invalid", Username: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	account, err = fixture.store.SetEmailVerified(ctx, account.SubjectID, account.AuthRevision)
	if err != nil {
		t.Fatal(err)
	}
	account, err = fixture.store.Activate(ctx, account.SubjectID, account.AuthRevision, []byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	return account
}

func authRequest(identifier, passwordValue string) AuthenticationRequest {
	return AuthenticationRequest{
		Identifier: identifier,
		Password:   []byte(passwordValue),
		Source:     throttle.ForwardedHeaders{RemoteAddr: "203.0.113.9:443"},
		DeviceKey:  "device-a",
	}
}

func TestProvisioningUsesOpaqueSubjectsAndPublicRegistrationIsDisabled(t *testing.T) {
	fixture := newTestStore(t)
	account, invitation, err := fixture.store.ProvisionInvitation(context.Background(), InvitationInput{Email: "Alice@Example.Invalid", Username: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSubjectID(account.SubjectID); err != nil || account.SubjectID == "Alice" {
		t.Fatalf("subject = %q/%v", account.SubjectID, err)
	}
	if account.State != AccountInvited || account.Email != "alice@example.invalid" || account.Username != "alice" || invitation.ExpiresAt.Sub(invitation.IssuedAt) != 24*time.Hour {
		t.Fatalf("provisioning snapshot = %+v invitation = %+v", account, invitation)
	}
	if err := fixture.store.RegisterPublic(context.Background(), InvitationInput{Email: "other@example.invalid"}); !errors.Is(err, ErrPublicRegistrationDisabled) {
		t.Fatalf("public registration error = %v", err)
	}
}

func TestInvitationVerificationIsSingleUseAndResendBounded(t *testing.T) {
	fixture := newTestStore(t)
	ctx := context.Background()
	account, invitation, err := fixture.store.ProvisionInvitation(ctx, InvitationInput{Email: "invite@example.invalid"})
	if err != nil || invitation.Token == "" {
		t.Fatalf("provision invitation = %+v/%v", invitation, err)
	}
	verified, err := fixture.store.VerifyInvitation(ctx, account.SubjectID, invitation.Token)
	if err != nil || !verified.EmailVerified || verified.AuthRevision != account.AuthRevision+1 {
		t.Fatalf("verify invitation = %+v/%v", verified, err)
	}
	if _, err := fixture.store.VerifyInvitation(ctx, account.SubjectID, invitation.Token); !errors.Is(err, ErrInvitationNotFound) {
		t.Fatalf("reused invitation = %v", err)
	}

	second, secondInvitation, err := fixture.store.ProvisionInvitation(ctx, InvitationInput{Email: "invite2@example.invalid"})
	if err != nil {
		t.Fatal(err)
	}
	for resend := 0; resend < 3; resend++ {
		secondInvitation, err = fixture.store.ResendInvitation(ctx, second.SubjectID)
		if err != nil || secondInvitation.Token == "" {
			t.Fatalf("resend %d = %+v/%v", resend, secondInvitation, err)
		}
	}
	if _, err := fixture.store.ResendInvitation(ctx, second.SubjectID); !errors.Is(err, ErrInvitationResendLimit) {
		t.Fatalf("fourth resend = %v", err)
	}
	if _, err := fixture.store.VerifyInvitation(ctx, second.SubjectID, invitation.Token); !errors.Is(err, ErrInvitationNotFound) {
		t.Fatalf("wrong invitation subject/token = %v", err)
	}
}

func TestIdentifierUniquenessAndStaleRevisionFailClosed(t *testing.T) {
	fixture := newTestStore(t)
	first, _, err := fixture.store.ProvisionInvitation(context.Background(), InvitationInput{Email: "same@example.invalid", Username: "same-user"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := fixture.store.ProvisionInvitation(context.Background(), InvitationInput{Email: "same@example.invalid", Username: "another-user"}); !errors.Is(err, ErrDuplicateIdentifier) {
		t.Fatalf("duplicate email error = %v", err)
	}
	if _, _, err := fixture.store.ProvisionInvitation(context.Background(), InvitationInput{Email: "other@example.invalid", Username: "same-user"}); !errors.Is(err, ErrDuplicateIdentifier) {
		t.Fatalf("duplicate username error = %v", err)
	}
	if _, err := fixture.store.SetEmailVerified(context.Background(), first.SubjectID, first.AuthRevision+1); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale verification error = %v", err)
	}
}

func TestAuthenticationUsesDummyHashAndChecksEmailAndAccountState(t *testing.T) {
	fixture := newTestStore(t)
	account := activateTestAccount(t, fixture)
	if _, err := fixture.store.Authenticate(context.Background(), authRequest("unknown@example.invalid", "wrong password")); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("unknown identifier result = %v", err)
	}
	if _, err := fixture.store.Authenticate(context.Background(), authRequest(account.Email, "wrong password")); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("wrong password result = %v", err)
	}
	result, err := fixture.store.Authenticate(context.Background(), authRequest(account.Email, "correct horse battery staple"))
	if err != nil || result.Account.SubjectID != account.SubjectID || !result.Account.CanIssueCredentials(*fixture.now) {
		t.Fatalf("successful authentication = %+v/%v", result, err)
	}
	if _, err := fixture.store.Transition(context.Background(), account.SubjectID, result.Account.AuthRevision, AccountSuspended); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.Authenticate(context.Background(), authRequest(account.Email, "correct horse battery staple")); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("suspended account result = %v", err)
	}
}

func TestFailedLoginsLockAccountWithoutPermanentAttackerLockout(t *testing.T) {
	fixture := newTestStore(t)
	account := activateTestAccount(t, fixture)
	for attempt := 0; attempt < 5; attempt++ {
		_, _ = fixture.store.Authenticate(context.Background(), authRequest(account.Email, "wrong password"))
	}
	snapshot, err := fixture.store.Snapshot(account.SubjectID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.State != AccountLocked || snapshot.LockedUntil.IsZero() {
		t.Fatalf("lockout snapshot = %+v", snapshot)
	}
	*fixture.now = snapshot.LockedUntil.Add(time.Second)
	result, err := fixture.store.Authenticate(context.Background(), authRequest(account.Email, "correct horse battery staple"))
	if err != nil || result.Account.State != AccountActive {
		t.Fatalf("expired lockout did not recover on valid authentication = %+v/%v", result, err)
	}
}

func TestPasswordChangeEnforcesCurrentPasswordHistoryAndRevision(t *testing.T) {
	fixture := newTestStore(t)
	account := activateTestAccount(t, fixture)
	changed, err := fixture.store.ChangePassword(context.Background(), account.SubjectID, account.AuthRevision, []byte("correct horse battery staple"), []byte("new correct password 2"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.ChangePassword(context.Background(), account.SubjectID, changed.AuthRevision, []byte("new correct password 2"), []byte("correct horse battery staple")); !errors.Is(err, password.ErrPasswordReused) {
		t.Fatalf("password history error = %v", err)
	}
	if _, err := fixture.store.ChangePassword(context.Background(), account.SubjectID, account.AuthRevision, []byte("new correct password 2"), []byte("another correct password 3")); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale password change error = %v", err)
	}
}

func TestConcurrentActivationAllowsOneRevisionWinner(t *testing.T) {
	fixture := newTestStore(t)
	account, _, err := fixture.store.ProvisionInvitation(context.Background(), InvitationInput{Email: "race@example.invalid"})
	if err != nil {
		t.Fatal(err)
	}
	account, err = fixture.store.SetEmailVerified(context.Background(), account.SubjectID, account.AuthRevision)
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 2)
	for _, value := range []string{"first correct password", "second correct password"} {
		wait.Add(1)
		go func(passwordValue string) {
			defer wait.Done()
			_, activationErr := fixture.store.Activate(context.Background(), account.SubjectID, account.AuthRevision, []byte(passwordValue))
			errorsSeen <- activationErr
		}(value)
	}
	wait.Wait()
	close(errorsSeen)
	var successCount, conflictCount int
	for activationErr := range errorsSeen {
		if activationErr == nil {
			successCount++
		} else if errors.Is(activationErr, ErrRevisionConflict) {
			conflictCount++
		}
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("activation outcomes success=%d conflict=%d", successCount, conflictCount)
	}
}
