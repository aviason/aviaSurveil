package mfa

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgreSQLStorePersistsEncryptedMFAState(t *testing.T) {
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
		t.Fatalf("apply auth migrations: %v", err)
	}
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	store, err := NewPostgresStore(pool, Config{EncryptionKey: []byte("01234567890123456789012345678901"), Clock: func() time.Time { return now }, Window: 1, MaxRecoveryFailures: 3})
	if err != nil {
		t.Fatal(err)
	}
	subjectID := "usr_MFADurableStateTest001"
	if _, err := pool.Exec(ctx, `INSERT INTO auth_identity.accounts(subject_id, state, email_verified, auth_revision, created_at, updated_at) VALUES ($1, 'invited', false, 1, $2, $2) ON CONFLICT (subject_id) DO NOTHING`, subjectID, now); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth_identity.accounts WHERE subject_id = $1`, subjectID)
	}()
	enrollment, err := store.Enroll(ctx, subjectID, "AviaSurveil360", "mfa@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	var ciphertext []byte
	if err := pool.QueryRow(ctx, `SELECT secret_ciphertext FROM auth_identity.mfa_factors WHERE subject_id = $1`, subjectID).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ciphertext), enrollment.Secret) {
		t.Fatal("MFA factor persisted the plaintext TOTP secret")
	}
	code, err := store.CurrentCodeForTesting(ctx, subjectID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ConfirmEnrollment(ctx, subjectID, code); err != nil {
		t.Fatalf("confirm MFA enrollment: %v", err)
	}
	if err := store.Verify(ctx, subjectID, code); !errors.Is(err, ErrCodeReplayed) {
		t.Fatalf("replayed MFA code = %v", err)
	}
	now = now.Add(30 * time.Second)
	code, err = store.CurrentCodeForTesting(ctx, subjectID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Verify(ctx, subjectID, code); err != nil {
		t.Fatalf("verify next MFA code: %v", err)
	}
	now = now.Add(30 * time.Second)
	code, err = store.CurrentCodeForTesting(ctx, subjectID, now)
	if err != nil {
		t.Fatal(err)
	}
	concurrentResults := make(chan error, 2)
	var concurrentVerifications sync.WaitGroup
	for range 2 {
		concurrentVerifications.Add(1)
		go func() {
			defer concurrentVerifications.Done()
			concurrentResults <- store.Verify(ctx, subjectID, code)
		}()
	}
	concurrentVerifications.Wait()
	close(concurrentResults)
	successes, replays := 0, 0
	for result := range concurrentResults {
		if result == nil {
			successes++
		}
		if errors.Is(result, ErrCodeReplayed) {
			replays++
		}
	}
	if successes != 1 || replays != 1 {
		t.Fatalf("concurrent MFA replay protection = successes:%d replays:%d", successes, replays)
	}
	codes, err := store.GenerateRecoveryCodes(ctx, subjectID, 3)
	if err != nil || len(codes) != 3 {
		t.Fatalf("generate recovery codes = %v/%v", codes, err)
	}
	var persistedHash []byte
	if err := pool.QueryRow(ctx, `SELECT code_hash FROM auth_identity.mfa_recovery_codes WHERE subject_id = $1 LIMIT 1`, subjectID).Scan(&persistedHash); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(persistedHash), codes[0]) {
		t.Fatal("MFA recovery code was persisted as plaintext")
	}
	if err := store.ConsumeRecoveryCode(ctx, subjectID, codes[0]); err != nil {
		t.Fatalf("consume recovery code: %v", err)
	}
	if err := store.ConsumeRecoveryCode(ctx, subjectID, codes[0]); !errors.Is(err, ErrRecoveryInvalid) {
		t.Fatalf("reused recovery code = %v", err)
	}
	if err := store.ConsumeRecoveryCode(ctx, subjectID, "invalid-one"); !errors.Is(err, ErrRecoveryInvalid) {
		t.Fatalf("first invalid recovery code = %v", err)
	}
	if err := store.ConsumeRecoveryCode(ctx, subjectID, "invalid-two"); !errors.Is(err, ErrRecoveryLocked) {
		t.Fatalf("bounded recovery failures = %v", err)
	}
	if snapshot, err := store.Snapshot(ctx, subjectID); err != nil || !snapshot.Enabled || snapshot.RecoveryCount != 2 || snapshot.LastUsedCounter < 0 {
		t.Fatalf("durable MFA snapshot = %+v/%v", snapshot, err)
	}
	if err := store.Reset(ctx, subjectID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(ctx, subjectID); !errors.Is(err, ErrFactorNotFound) {
		t.Fatalf("MFA reset snapshot = %v", err)
	}
}
