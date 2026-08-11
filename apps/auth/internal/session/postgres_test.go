package session

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

func TestPostgreSQLSessionRotationAndReuse(t *testing.T) {
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
	clock := &sessionTestClock{now: time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC)}
	authorizer := &sessionTestAuthorizer{authorized: true}
	store, err := NewPostgresStore(pool, Config{
		Authorizer:            authorizer,
		Clients:               StaticClientRegistry{"web": {"https://web.example/callback": {}}},
		FingerprintKey:        []byte("01234567890123456789012345678901"),
		Clock:                 clock.Now,
		IdleTTL:               15 * time.Minute,
		AbsoluteTTL:           time.Hour,
		MaxFamiliesPerSubject: 2,
	})
	if err != nil {
		t.Fatalf("new PostgreSQL session store: %v", err)
	}
	subjectID := "usr_" + strings.Repeat("P", 22)
	cleanup := func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_identity.refresh_token_history WHERE family_id IN (SELECT family_id FROM auth_identity.provider_sessions WHERE subject_id = $1)`, subjectID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_identity.refresh_families WHERE session_id IN (SELECT session_id FROM auth_identity.provider_sessions WHERE subject_id = $1)`, subjectID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_identity.provider_sessions WHERE subject_id = $1`, subjectID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_identity.accounts WHERE subject_id = $1`, subjectID)
	}
	defer cleanup()
	if _, err := pool.Exec(ctx, `INSERT INTO auth_identity.accounts(subject_id, state, password_hash, email_verified, auth_revision, created_at, updated_at) VALUES ($1, 'active', 'argon2id-test', true, 7, $2, $2)`, subjectID, clock.Now()); err != nil {
		t.Fatalf("insert disposable account: %v", err)
	}

	issued, err := store.Issue(ctx, IssueInput{
		SubjectID: subjectID, AuthRevision: 7, ClientID: "web", RedirectURI: "https://web.example/callback",
		Fingerprint: FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"},
	})
	if err != nil {
		t.Fatalf("issue durable session: %v", err)
	}
	rotated, err := store.Rotate(ctx, rotateInput(issued))
	if err != nil {
		t.Fatalf("rotate durable session: %v", err)
	}
	if rotated.Generation != 2 || rotated.RefreshToken == issued.RefreshToken {
		t.Fatalf("unexpected durable rotation: %+v", rotated)
	}
	if _, err := store.Rotate(ctx, rotateInput(issued)); !errors.Is(err, ErrRefreshReuse) {
		t.Fatalf("reused durable token error = %v, want refresh reuse", err)
	}
	snapshot, err := store.Snapshot(ctx, issued.FamilyID)
	if err != nil {
		t.Fatalf("read reused durable family: %v", err)
	}
	if snapshot.FamilyState != FamilyReuseDetected || snapshot.SessionState != SessionRevoked {
		t.Fatalf("durable reuse did not revoke family/session: %+v", snapshot)
	}

	second, err := store.Issue(ctx, IssueInput{
		SubjectID: subjectID, AuthRevision: 7, ClientID: "web", RedirectURI: "https://web.example/callback",
		Fingerprint: FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"},
	})
	if err != nil {
		t.Fatalf("issue second durable session: %v", err)
	}
	third, err := store.Issue(ctx, IssueInput{
		SubjectID: subjectID, AuthRevision: 7, ClientID: "web", RedirectURI: "https://web.example/callback",
		Fingerprint: FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"},
	})
	if err != nil {
		t.Fatalf("issue third durable session: %v", err)
	}
	if _, err := store.Issue(ctx, IssueInput{
		SubjectID: subjectID, AuthRevision: 7, ClientID: "web", RedirectURI: "https://web.example/callback",
		Fingerprint: FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"},
	}); err != nil {
		t.Fatalf("issue bounded durable session: %v", err)
	}
	oldest, err := store.Snapshot(ctx, second.FamilyID)
	if err != nil {
		t.Fatalf("read bounded durable family: %v", err)
	}
	if oldest.FamilyState != FamilyRevoked {
		t.Fatalf("bounded durable family state = %s, want revoked", oldest.FamilyState)
	}
	if _, err := store.Rotate(ctx, rotateInput(third)); err != nil {
		t.Fatalf("rotate bounded durable family: %v", err)
	}

	// A database row lock, rather than an application mutex, must serialize
	// concurrent presentations of one current token.
	fourth, err := store.Issue(ctx, IssueInput{
		SubjectID: subjectID, AuthRevision: 7, ClientID: "web", RedirectURI: "https://web.example/callback",
		Fingerprint: FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"},
	})
	if err != nil {
		t.Fatalf("issue concurrent durable session: %v", err)
	}
	const attempts = 8
	results := make(chan error, attempts)
	var wait sync.WaitGroup
	wait.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			defer wait.Done()
			_, rotateErr := store.Rotate(ctx, rotateInput(fourth))
			results <- rotateErr
		}()
	}
	wait.Wait()
	close(results)
	success, reuse := 0, 0
	for rotateErr := range results {
		if rotateErr == nil {
			success++
		} else if errors.Is(rotateErr, ErrRefreshReuse) {
			reuse++
		}
	}
	if success != 1 || reuse != attempts-1 {
		t.Fatalf("durable concurrent rotation success/reuse = %d/%d, want 1/%d", success, reuse, attempts-1)
	}
}

func TestPostgreSQLSessionExpiryAndRevocation(t *testing.T) {
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
	clock := &sessionTestClock{now: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)}
	authorizer := &sessionTestAuthorizer{authorized: true}
	store, err := NewPostgresStore(pool, Config{
		Authorizer:            authorizer,
		Clients:               StaticClientRegistry{"web": {"https://web.example/callback": {}}},
		FingerprintKey:        []byte("01234567890123456789012345678901"),
		Clock:                 clock.Now,
		IdleTTL:               10 * time.Minute,
		AbsoluteTTL:           30 * time.Minute,
		MaxFamiliesPerSubject: 2,
	})
	if err != nil {
		t.Fatalf("new PostgreSQL session store: %v", err)
	}
	subjectID := "usr_" + strings.Repeat("E", 22)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_identity.refresh_token_history WHERE family_id IN (SELECT family_id FROM auth_identity.provider_sessions WHERE subject_id = $1)`, subjectID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_identity.refresh_families WHERE session_id IN (SELECT session_id FROM auth_identity.provider_sessions WHERE subject_id = $1)`, subjectID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_identity.provider_sessions WHERE subject_id = $1`, subjectID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_identity.accounts WHERE subject_id = $1`, subjectID)
	}()
	if _, err := pool.Exec(ctx, `INSERT INTO auth_identity.accounts(subject_id, state, password_hash, email_verified, auth_revision, created_at, updated_at) VALUES ($1, 'active', 'argon2id-test', true, 7, $2, $2)`, subjectID, clock.Now()); err != nil {
		t.Fatalf("insert disposable account: %v", err)
	}
	issued, err := store.Issue(ctx, IssueInput{
		SubjectID: subjectID, AuthRevision: 7, ClientID: "web", RedirectURI: "https://web.example/callback",
		Fingerprint: FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"},
	})
	if err != nil {
		t.Fatalf("issue expiring durable session: %v", err)
	}
	clock.Advance(31 * time.Minute)
	if cleaned, cleanupErr := store.Cleanup(ctx, clock.Now()); cleanupErr != nil || cleaned != 1 {
		t.Fatalf("cleanup = %d/%v, want 1/nil", cleaned, cleanupErr)
	}
	if _, err := store.Rotate(ctx, rotateInput(issued)); !errors.Is(err, ErrRefreshExpired) {
		t.Fatalf("expired durable token error = %v, want refresh expired", err)
	}

	second, err := store.Issue(ctx, IssueInput{
		SubjectID: subjectID, AuthRevision: 7, ClientID: "web", RedirectURI: "https://web.example/callback",
		Fingerprint: FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"},
	})
	if err != nil {
		t.Fatalf("issue revocable durable session: %v", err)
	}
	if count, revokeErr := store.RevokeAllSessions(ctx, subjectID); revokeErr != nil || count != 1 {
		t.Fatalf("revoke all = %d/%v, want 1/nil", count, revokeErr)
	}
	if _, err := store.Rotate(ctx, rotateInput(second)); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("revoked durable token error = %v, want session revoked", err)
	}
}
