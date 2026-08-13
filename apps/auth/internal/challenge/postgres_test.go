package challenge

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

func TestPostgreSQLStorePersistsBoundedChallenges(t *testing.T) {
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
	now := time.Date(2026, 8, 11, 17, 0, 0, 0, time.UTC)
	store, err := NewPostgresStore(pool, Config{Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	subjectID := "usr_" + strings.Repeat("C", 22)
	if _, err := pool.Exec(ctx, `INSERT INTO auth_identity.accounts(subject_id, state, email_verified, auth_revision, created_at, updated_at) VALUES ($1, 'invited', false, 1, $2, $2) ON CONFLICT (subject_id) DO NOTHING`, subjectID, now); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth_identity.accounts WHERE subject_id = $1`, subjectID)
	}()

	issued, err := store.Issue(ctx, subjectID, PurposePasswordReset, time.Hour, 2)
	if err != nil {
		t.Fatal(err)
	}
	var tokenHash []byte
	if err := pool.QueryRow(ctx, `SELECT token_hash FROM auth_identity.identity_challenges WHERE subject_id = $1 AND purpose = $2`, subjectID, PurposePasswordReset).Scan(&tokenHash); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(tokenHash), issued.Token) {
		t.Fatal("challenge token was persisted as plaintext")
	}
	if err := store.Consume(ctx, subjectID, PurposeMFARecovery, issued.Token); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("wrong purpose consumption = %v", err)
	}
	if err := store.RejectAttempt(ctx, subjectID, issued.Purpose, issued.Token); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("first rejected attempt = %v", err)
	}
	if err := store.RejectAttempt(ctx, subjectID, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeLocked) {
		t.Fatalf("bounded rejected attempt = %v", err)
	}
	if err := store.Consume(ctx, subjectID, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeLocked) {
		t.Fatalf("locked challenge consumption = %v", err)
	}

	issued, err = store.Issue(ctx, subjectID, PurposeEmailVerification, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Consume(ctx, subjectID, issued.Purpose, issued.Token); err != nil {
		t.Fatalf("consume challenge: %v", err)
	}
	if err := store.Consume(ctx, subjectID, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeUsed) {
		t.Fatalf("reused challenge = %v", err)
	}
	issued, err = store.Issue(ctx, subjectID, PurposeEmailVerification, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	concurrentResults := make(chan error, 2)
	var concurrentConsumes sync.WaitGroup
	for range 2 {
		concurrentConsumes.Add(1)
		go func() {
			defer concurrentConsumes.Done()
			concurrentResults <- store.Consume(ctx, subjectID, issued.Purpose, issued.Token)
		}()
	}
	concurrentConsumes.Wait()
	close(concurrentResults)
	successes, used := 0, 0
	for result := range concurrentResults {
		if result == nil {
			successes++
		}
		if errors.Is(result, ErrChallengeUsed) {
			used++
		}
	}
	if successes != 1 || used != 1 {
		t.Fatalf("concurrent challenge consumption = successes:%d used:%d", successes, used)
	}

	issued, err = store.Issue(ctx, subjectID, PurposeAdminRecovery, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	if invalidated, err := store.Invalidate(ctx, subjectID, issued.Purpose); err != nil || invalidated != 1 {
		t.Fatalf("invalidate challenge = %d/%v", invalidated, err)
	}
	if err := store.Consume(ctx, subjectID, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeUsed) {
		t.Fatalf("invalidated challenge consumption = %v", err)
	}

	issued, err = store.Issue(ctx, subjectID, PurposeMFARecovery, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour)
	if err := store.Consume(ctx, subjectID, issued.Purpose, issued.Token); !errors.Is(err, ErrChallengeExpired) {
		t.Fatalf("expired challenge consumption = %v", err)
	}
	if removed, err := store.Cleanup(ctx, now, 5); err != nil || removed < 1 || removed > 5 {
		t.Fatalf("cleanup challenges = %d/%v", removed, err)
	}
}
