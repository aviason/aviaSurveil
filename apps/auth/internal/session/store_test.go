package session

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type sessionTestClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *sessionTestClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *sessionTestClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(duration)
	clock.mu.Unlock()
}

type sessionTestAuthorizer struct {
	mu           sync.Mutex
	authorized   bool
	lastSubject  string
	lastRevision uint64
}

func (authorizer *sessionTestAuthorizer) Authorize(_ context.Context, subjectID string, revision uint64) (bool, error) {
	authorizer.mu.Lock()
	defer authorizer.mu.Unlock()
	authorizer.lastSubject = subjectID
	authorizer.lastRevision = revision
	return authorizer.authorized, nil
}

func newSessionTestStore(t *testing.T) (*Store, *sessionTestClock, *sessionTestAuthorizer) {
	t.Helper()
	clock := &sessionTestClock{now: time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)}
	authorizer := &sessionTestAuthorizer{authorized: true}
	store, err := NewStore(Config{
		Authorizer:            authorizer,
		Clients:               StaticClientRegistry{"web": {"https://web.example/callback": {}}},
		FingerprintKey:        []byte("01234567890123456789012345678901"),
		Clock:                 clock.Now,
		IdleTTL:               15 * time.Minute,
		AbsoluteTTL:           time.Hour,
		MaxFamiliesPerSubject: 2,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store, clock, authorizer
}

func issueSession(t *testing.T, store *Store) IssuedRefresh {
	t.Helper()
	issued, err := store.Issue(context.Background(), IssueInput{
		SubjectID: "usr_0123456789012345678901", AuthRevision: 7,
		ClientID: "web", RedirectURI: "https://web.example/callback",
		Fingerprint: FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"},
	})
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	return issued
}

func rotateInput(issued IssuedRefresh) RotateInput {
	return RotateInput{
		RefreshToken: issued.RefreshToken, ClientID: "web", RedirectURI: "https://web.example/callback",
		Fingerprint: FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"},
	}
}

func TestIssueAndRotateBindsClientFingerprintAndRevision(t *testing.T) {
	store, _, authorizer := newSessionTestStore(t)
	issued := issueSession(t, store)
	if len(issued.RefreshToken) != 43 || issued.SessionID[:4] != "ses_" || issued.FamilyID[:4] != "fam_" {
		t.Fatalf("unexpected issued identifiers/token: %+v", issued)
	}
	rotated, err := store.Rotate(context.Background(), rotateInput(issued))
	if err != nil {
		t.Fatalf("rotate session: %v", err)
	}
	if rotated.RefreshToken == issued.RefreshToken || rotated.Generation != 2 {
		t.Fatalf("rotation did not replace token/generation: %+v", rotated)
	}
	authorizer.mu.Lock()
	if authorizer.lastSubject != "usr_0123456789012345678901" || authorizer.lastRevision != 7 {
		t.Fatalf("authorizer binding lost: %s/%d", authorizer.lastSubject, authorizer.lastRevision)
	}
	authorizer.mu.Unlock()

	if _, err := store.Rotate(context.Background(), rotateInput(issued)); !errors.Is(err, ErrRefreshReuse) {
		t.Fatalf("old token error = %v, want refresh reuse", err)
	}
	snapshot, err := store.Snapshot(issued.FamilyID)
	if err != nil {
		t.Fatalf("snapshot after reuse: %v", err)
	}
	if snapshot.FamilyState != FamilyReuseDetected || snapshot.SessionState != SessionRevoked {
		t.Fatalf("reuse did not revoke family/session: %+v", snapshot)
	}
}

func TestRotateRejectsFingerprintAndClientAndRevokesFamily(t *testing.T) {
	store, _, _ := newSessionTestStore(t)
	issued := issueSession(t, store)
	wrongFingerprint := rotateInput(issued)
	wrongFingerprint.Fingerprint.UserAgent = "different-browser"
	if _, err := store.Rotate(context.Background(), wrongFingerprint); !errors.Is(err, ErrFingerprintMismatch) {
		t.Fatalf("wrong fingerprint error = %v", err)
	}
	if _, err := store.Rotate(context.Background(), rotateInput(issued)); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("revoked family error = %v, want session revoked", err)
	}

	second := issueSession(t, store)
	wrongClient := rotateInput(second)
	wrongClient.ClientID = "other"
	if _, err := store.Rotate(context.Background(), wrongClient); !errors.Is(err, ErrInvalidClient) {
		t.Fatalf("wrong client error = %v, want invalid client", err)
	}
}

func TestRotateRejectsStaleAuthorizationAndExpiry(t *testing.T) {
	store, clock, authorizer := newSessionTestStore(t)
	issued := issueSession(t, store)
	authorizer.mu.Lock()
	authorizer.authorized = false
	authorizer.mu.Unlock()
	if _, err := store.Rotate(context.Background(), rotateInput(issued)); !errors.Is(err, ErrAuthRevisionStale) {
		t.Fatalf("stale authorization error = %v", err)
	}

	authorizer.mu.Lock()
	authorizer.authorized = true
	authorizer.mu.Unlock()
	second := issueSession(t, store)
	clock.Advance(time.Hour)
	if expired := store.Cleanup(clock.Now()); expired < 1 {
		t.Fatalf("cleanup expired %d families, want at least one", expired)
	}
	if _, err := store.Rotate(context.Background(), rotateInput(second)); !errors.Is(err, ErrRefreshExpired) {
		t.Fatalf("expired rotation error = %v, want refresh expired", err)
	}
}

func TestIssueBoundsActiveFamiliesPerSubject(t *testing.T) {
	store, clock, _ := newSessionTestStore(t)
	first := issueSession(t, store)
	clock.Advance(time.Second)
	second := issueSession(t, store)
	clock.Advance(time.Second)
	third := issueSession(t, store)
	firstSnapshot, err := store.Snapshot(first.FamilyID)
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	if firstSnapshot.FamilyState != FamilyRevoked {
		t.Fatalf("oldest family state = %s, want revoked", firstSnapshot.FamilyState)
	}
	for _, familyID := range []string{second.FamilyID, third.FamilyID} {
		snapshot, snapshotErr := store.Snapshot(familyID)
		if snapshotErr != nil {
			t.Fatalf("snapshot %s: %v", familyID, snapshotErr)
		}
		if snapshot.FamilyState != FamilyActive {
			t.Fatalf("bounded family %s state = %s, want active", familyID, snapshot.FamilyState)
		}
	}
}

func TestConcurrentRotateAllowsExactlyOneUse(t *testing.T) {
	store, _, _ := newSessionTestStore(t)
	issued := issueSession(t, store)
	const attempts = 16
	results := make(chan error, attempts)
	var wait sync.WaitGroup
	wait.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			defer wait.Done()
			_, err := store.Rotate(context.Background(), rotateInput(issued))
			results <- err
		}()
	}
	wait.Wait()
	close(results)
	var success, reuse int
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrRefreshReuse):
			reuse++
		}
	}
	if success != 1 || reuse != attempts-1 {
		t.Fatalf("concurrent rotation success/reuse = %d/%d, want 1/%d", success, reuse, attempts-1)
	}
	snapshot, err := store.Snapshot(issued.FamilyID)
	if err != nil {
		t.Fatalf("snapshot after concurrent rotation: %v", err)
	}
	if snapshot.FamilyState != FamilyReuseDetected {
		t.Fatalf("concurrent reuse family state = %s, want reuse-detected", snapshot.FamilyState)
	}
}

func TestRevokeAllAndFingerprintHashDoNotRetainRawInputs(t *testing.T) {
	store, _, _ := newSessionTestStore(t)
	first := issueSession(t, store)
	second := issueSession(t, store)
	if revoked := store.RevokeAll(context.Background(), "usr_0123456789012345678901"); revoked != 2 {
		t.Fatalf("revoke all = %d, want 2", revoked)
	}
	if _, err := store.Rotate(context.Background(), rotateInput(first)); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("revoked first rotation error = %v", err)
	}
	if _, err := store.Rotate(context.Background(), rotateInput(second)); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("revoked second rotation error = %v", err)
	}
	fingerprint := DeriveFingerprint([]byte("01234567890123456789012345678901"), FingerprintInput{UserAgent: "browser/1", ClientIP: "192.0.2.10"})
	if string(fingerprint[:]) == "browser/1" || string(fingerprint[:]) == "192.0.2.10" {
		t.Fatal("fingerprint unexpectedly contains raw input")
	}
}
