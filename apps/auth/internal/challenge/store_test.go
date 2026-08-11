package challenge

import (
	"errors"
	"testing"
	"time"
)

func TestChallengesAreSingleUseExpiringAndAttemptBounded(t *testing.T) {
	now := time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC)
	store := NewStore(Config{Clock: func() time.Time { return now }})
	issued, err := store.Issue("usr_0123456789012345678901", PurposePasswordReset, time.Hour, 2)
	if err != nil || issued.Token == "" {
		t.Fatalf("issue challenge = %+v/%v", issued, err)
	}
	if err := store.RejectAttempt(issued.Subject, issued.Purpose, issued.Token); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("first rejection = %v", err)
	}
	if err := store.RejectAttempt(issued.Subject, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeLocked) {
		t.Fatalf("second rejection = %v", err)
	}
	if err := store.Consume(issued.Subject, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeLocked) {
		t.Fatalf("locked consume = %v", err)
	}

	issued, err = store.Issue("usr_0123456789012345678901", PurposeEmailVerification, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Consume(issued.Subject, issued.Purpose, issued.Token); err != nil {
		t.Fatalf("consume challenge: %v", err)
	}
	if err := store.Consume(issued.Subject, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeUsed) {
		t.Fatalf("reused challenge = %v", err)
	}

	issued, err = store.Issue("usr_0123456789012345678901", PurposeAdminRecovery, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour)
	if err := store.Consume(issued.Subject, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeExpired) {
		t.Fatalf("expired challenge = %v", err)
	}
	if removed := store.Cleanup(now); removed < 2 {
		t.Fatalf("cleanup removed %d challenges, want at least 2", removed)
	}
}

func TestChallengeSubjectAndPurposeAreBound(t *testing.T) {
	store := NewStore(Config{})
	issued, err := store.Issue("usr_0123456789012345678901", PurposeMFARecovery, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Consume("usr_other", issued.Purpose, issued.Token); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("wrong subject = %v", err)
	}
	if err := store.Consume(issued.Subject, PurposePasswordReset, issued.Token); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("wrong purpose = %v", err)
	}
	if invalid := store.Invalidate(issued.Subject, issued.Purpose); invalid != 1 {
		t.Fatalf("invalidated count = %d", invalid)
	}
	if err := store.Consume(issued.Subject, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeUsed) {
		t.Fatalf("invalidated challenge = %v", err)
	}
}
