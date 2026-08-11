package mfa

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func newMFATestStore(t *testing.T) (*Store, *time.Time) {
	t.Helper()
	now := time.Date(2026, 8, 11, 13, 0, 0, 0, time.UTC)
	store, err := NewStore(Config{EncryptionKey: []byte("01234567890123456789012345678901"), Clock: func() time.Time { return now }, Window: 1, MaxRecoveryFailures: 3})
	if err != nil {
		t.Fatalf("new MFA store: %v", err)
	}
	return store, &now
}

func TestTOTPEnrollmentProtectionAndReplayWindow(t *testing.T) {
	store, now := newMFATestStore(t)
	enrollment, err := store.Enroll("usr_0123456789012345678901", "AviaSurveil360", "pilot@example.invalid")
	if err != nil {
		t.Fatalf("enroll TOTP: %v", err)
	}
	if enrollment.Secret == "" || !strings.HasPrefix(enrollment.OTPAuthURI, "otpauth://totp/") || strings.Contains(enrollment.OTPAuthURI, " ") {
		t.Fatalf("enrollment = %+v", enrollment)
	}
	if err := store.ConfirmEnrollment(enrollment.SubjectID, mustCode(t, store, enrollment.SubjectID, *now)); err != nil {
		t.Fatalf("confirm TOTP: %v", err)
	}
	if err := store.Verify(enrollment.SubjectID, mustCode(t, store, enrollment.SubjectID, *now)); !errors.Is(err, ErrCodeReplayed) {
		t.Fatalf("replayed current TOTP = %v, want replay denial", err)
	}
	*now = now.Add(30 * time.Second)
	if err := store.Verify(enrollment.SubjectID, mustCode(t, store, enrollment.SubjectID, *now)); err != nil {
		t.Fatalf("verify next TOTP step: %v", err)
	}
	snapshot, err := store.Snapshot(enrollment.SubjectID)
	if err != nil || !snapshot.Enabled || snapshot.LastUsedCounter < 0 {
		t.Fatalf("MFA snapshot = %+v/%v", snapshot, err)
	}
	if strings.Contains(string(store.factors[enrollment.SubjectID].secretCiphertext), enrollment.Secret) {
		t.Fatal("stored MFA ciphertext contains raw secret")
	}
}

func TestRecoveryCodesAreHashedSingleUseAndBounded(t *testing.T) {
	store, now := newMFATestStore(t)
	enrollment, err := store.Enroll("usr_0123456789012345678901", "AviaSurveil360", "pilot@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ConfirmEnrollment(enrollment.SubjectID, mustCode(t, store, enrollment.SubjectID, *now)); err != nil {
		t.Fatal(err)
	}
	codes, err := store.GenerateRecoveryCodes(enrollment.SubjectID, 3)
	if err != nil || len(codes) != 3 {
		t.Fatalf("recovery generation = %v/%v", codes, err)
	}
	if err := store.ConsumeRecoveryCode(enrollment.SubjectID, codes[0]); err != nil {
		t.Fatalf("consume recovery code: %v", err)
	}
	if err := store.ConsumeRecoveryCode(enrollment.SubjectID, codes[0]); !errors.Is(err, ErrRecoveryInvalid) {
		t.Fatalf("reused recovery code = %v, want generic invalid", err)
	}
	if err := store.ConsumeRecoveryCode(enrollment.SubjectID, "wrong-code"); !errors.Is(err, ErrRecoveryInvalid) {
		t.Fatalf("wrong recovery code = %v", err)
	}
	if err := store.ConsumeRecoveryCode(enrollment.SubjectID, "wrong-code-2"); !errors.Is(err, ErrRecoveryLocked) {
		t.Fatalf("third recovery failure = %v, want lock", err)
	}
	if err := store.ConsumeRecoveryCode(enrollment.SubjectID, codes[1]); !errors.Is(err, ErrRecoveryLocked) {
		t.Fatalf("locked recovery code = %v", err)
	}
	if snapshot, snapshotErr := store.Snapshot(enrollment.SubjectID); snapshotErr != nil || snapshot.RecoveryCount != 2 {
		t.Fatalf("recovery snapshot = %+v/%v", snapshot, snapshotErr)
	}
}

func TestMFAResetAndPolicyBounds(t *testing.T) {
	if _, err := NewStore(Config{EncryptionKey: []byte("short")}); err == nil {
		t.Fatal("short MFA key accepted")
	}
	store, now := newMFATestStore(t)
	enrollment, err := store.Enroll("usr_0123456789012345678901", "AviaSurveil360", "pilot@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Verify(enrollment.SubjectID, "000000"); !errors.Is(err, ErrFactorDisabled) {
		t.Fatalf("unconfirmed factor verify = %v", err)
	}
	if err := store.Reset(enrollment.SubjectID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(enrollment.SubjectID); !errors.Is(err, ErrFactorNotFound) {
		t.Fatalf("reset snapshot = %v", err)
	}
	_ = now
}

func mustCode(t *testing.T, store *Store, subjectID string, at time.Time) string {
	t.Helper()
	code, err := store.CurrentCodeForTesting(subjectID, at)
	if err != nil {
		t.Fatalf("current test code: %v", err)
	}
	return code
}
