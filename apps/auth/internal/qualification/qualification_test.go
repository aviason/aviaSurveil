package qualification

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/challenge"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mfa"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/password"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/session"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
)

// qualificationManifest is intentionally synthetic and is never used as a
// production identity source.
//
//go:embed testdata/qualification-manifest.json
var qualificationManifest []byte

type manifest struct {
	Profile    string            `json:"profile"`
	Issuer     string            `json:"issuer"`
	ClientID   string            `json:"client_id"`
	Database   string            `json:"database"`
	Cookie     string            `json:"cookie_prefix"`
	Ports      map[string]int    `json:"ports"`
	KeyIDs     map[string]string `json:"key_ids"`
	Identities []struct {
		Subject      string `json:"subject"`
		Email        string `json:"email"`
		Organization string `json:"organization"`
		Role         string `json:"role"`
	} `json:"identities"`
}

const (
	syntheticSubjectEmail = "synthetic-one@example.invalid"
	syntheticSubjectName  = "synthetic-one"
	clientID              = "as360-web-candidate"
	redirectURI           = "https://web.candidate.invalid/auth/callback"
)

type membershipAuthorizer struct {
	mu          sync.RWMutex
	identities  *identity.Store
	memberships map[string]bool
}

func (authorizer *membershipAuthorizer) Authorize(_ context.Context, subjectID string, revision uint64) (bool, error) {
	snapshot, err := authorizer.identities.Snapshot(subjectID)
	if err != nil {
		return false, err
	}
	authorizer.mu.RLock()
	member := authorizer.memberships[subjectID]
	authorizer.mu.RUnlock()
	return member && snapshot.AuthRevision == revision && snapshot.CanIssueCredentials(time.Now().UTC()), nil
}

func (authorizer *membershipAuthorizer) setMembership(subjectID string, enabled bool) {
	authorizer.mu.Lock()
	defer authorizer.mu.Unlock()
	authorizer.memberships[subjectID] = enabled
}

type revokerBridge struct{ store *session.Store }

func (bridge *revokerBridge) RevokeAllSessions(ctx context.Context, subjectID string) error {
	if bridge.store == nil {
		return errors.New("session store is not initialized")
	}
	return bridge.store.RevokeAllSessions(ctx, subjectID)
}

func TestSyntheticQualificationManifestFreezesDistinctTrustBoundaries(t *testing.T) {
	var fixture manifest
	if err := json.Unmarshal(qualificationManifest, &fixture); err != nil {
		t.Fatalf("decode qualification manifest: %v", err)
	}
	if fixture.Profile != "isolated-candidate" || fixture.Issuer == "" || fixture.ClientID == "" || fixture.Database == "" || fixture.Cookie == "" {
		t.Fatalf("incomplete isolated qualification boundary: %+v", fixture)
	}
	if fixture.Ports["http"] == 0 || fixture.Ports["postgres"] == 0 || fixture.Ports["mailpit"] == 0 || fixture.KeyIDs["signing"] == "" || fixture.KeyIDs["mfa"] == "" {
		t.Fatalf("qualification manifest omitted an isolated resource: %+v", fixture)
	}
	seen := make(map[string]struct{}, len(fixture.Identities))
	for _, entry := range fixture.Identities {
		if err := identity.ValidateSubjectID(entry.Subject); err != nil {
			t.Fatalf("manifest subject %q is not an opaque subject: %v", entry.Subject, err)
		}
		if _, exists := seen[entry.Subject]; exists {
			t.Fatalf("manifest subject is duplicated: %q", entry.Subject)
		}
		seen[entry.Subject] = struct{}{}
		if entry.Subject == entry.Email || strings.Contains(entry.Subject, entry.Email) || entry.Organization == "" || entry.Role == "" {
			t.Fatalf("manifest subject is derived from mutable identity data: %+v", entry)
		}
	}
}

func TestSyntheticCandidateQualificationKeepsMembershipOutsideIdentityAuthority(t *testing.T) {
	ctx := context.Background()
	hasher, err := password.New(password.Params{
		MemoryKiB: 16 * 1024, Time: 1, Threads: 1, KeyLength: 16, SaltLen: 16,
		MaxBytes: 1024, Capacity: 2,
	})
	if err != nil {
		t.Fatalf("new bounded test hasher: %v", err)
	}
	limiter, err := throttle.NewMemoryLimiter(time.Minute, 20, nil)
	if err != nil {
		t.Fatalf("new limiter: %v", err)
	}
	var revoker revokerBridge
	identities, err := identity.NewStore(identity.Config{
		Hasher: hasher, PasswordPolicy: password.Policy{MinBytes: 12, MaxBytes: 1024},
		Limiter: limiter, SessionRevoker: &revoker, TrustedProxies: []netip.Prefix{},
	})
	if err != nil {
		t.Fatalf("new identity store: %v", err)
	}
	authorizer := &membershipAuthorizer{identities: identities, memberships: make(map[string]bool)}
	sessions, err := session.NewStore(session.Config{
		Authorizer:     authorizer,
		Clients:        session.StaticClientRegistry{clientID: {redirectURI: {}}},
		FingerprintKey: []byte("qualification-fingerprint-key-2026-08-11"),
		IdleTTL:        15 * time.Minute, AbsoluteTTL: 8 * time.Hour, MaxFamiliesPerSubject: 2,
	})
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	revoker.store = sessions

	account, invitation, err := identities.ProvisionInvitation(ctx, identity.InvitationInput{Email: syntheticSubjectEmail, Username: syntheticSubjectName})
	if err != nil {
		t.Fatalf("provision synthetic invitation: %v", err)
	}
	if !strings.HasPrefix(account.SubjectID, "usr_") || account.SubjectID == syntheticSubjectEmail || strings.Contains(account.SubjectID, syntheticSubjectName) {
		t.Fatalf("subject must be opaque and independent of mutable identifiers: %q", account.SubjectID)
	}
	verified, err := identities.VerifyInvitation(ctx, account.SubjectID, invitation.Token)
	if err != nil {
		t.Fatalf("verify invitation: %v", err)
	}
	if _, err := identities.VerifyInvitation(ctx, account.SubjectID, invitation.Token); !errors.Is(err, identity.ErrInvitationNotFound) {
		t.Fatalf("replayed invitation error = %v", err)
	}
	active, err := identities.Activate(ctx, account.SubjectID, verified.AuthRevision, []byte("synthetic-password-2026"))
	if err != nil {
		t.Fatalf("activate synthetic identity: %v", err)
	}
	if !active.CanIssueCredentials(time.Now().UTC()) {
		t.Fatal("activated verified identity should be credential-eligible")
	}

	// An authenticated provider identity without application membership cannot
	// mint a refresh family. Membership remains an Avia application decision.
	issueInput := session.IssueInput{SubjectID: active.SubjectID, AuthRevision: active.AuthRevision, ClientID: clientID, RedirectURI: redirectURI, Fingerprint: session.FingerprintInput{UserAgent: "qualification", ClientIP: "192.0.2.10"}}
	if _, err := sessions.Issue(ctx, issueInput); !errors.Is(err, session.ErrSubjectUnauthorized) {
		t.Fatalf("bootstrap identity without membership issued authority: %v", err)
	}
	authorizer.setMembership(active.SubjectID, true)
	issued, err := sessions.Issue(ctx, issueInput)
	if err != nil {
		t.Fatalf("member identity should receive provider session: %v", err)
	}

	mfaStore, err := mfa.NewStore(mfa.Config{EncryptionKey: []byte("0123456789abcdef0123456789abcdef"), Window: 1})
	if err != nil {
		t.Fatalf("new MFA store: %v", err)
	}
	enrollment, err := mfaStore.Enroll(active.SubjectID, "https://auth.candidate.invalid", syntheticSubjectEmail)
	if err != nil || strings.Contains(enrollment.OTPAuthURI, "0123456789abcdef") {
		t.Fatalf("MFA enrollment failed or leaked key material: %v", err)
	}
	code, err := mfaStore.CurrentCodeForTesting(active.SubjectID, time.Now())
	if err != nil {
		t.Fatalf("derive test TOTP: %v", err)
	}
	if err := mfaStore.ConfirmEnrollment(active.SubjectID, code); err != nil {
		t.Fatalf("confirm TOTP enrollment: %v", err)
	}
	if err := mfaStore.Verify(active.SubjectID, code); !errors.Is(err, mfa.ErrCodeReplayed) {
		t.Fatalf("reused enrollment code should be rejected: %v", err)
	}
	recoveryCodes, err := mfaStore.GenerateRecoveryCodes(active.SubjectID, 2)
	if err != nil || len(recoveryCodes) != 2 {
		t.Fatalf("generate recovery codes: %v", err)
	}
	if err := mfaStore.ConsumeRecoveryCode(active.SubjectID, recoveryCodes[0]); err != nil {
		t.Fatalf("consume recovery code: %v", err)
	}
	if err := mfaStore.ConsumeRecoveryCode(active.SubjectID, recoveryCodes[0]); !errors.Is(err, mfa.ErrRecoveryInvalid) {
		t.Fatalf("reused recovery code should be rejected: %v", err)
	}

	challenges := challenge.NewStore(challenge.Config{})
	issuedChallenge, err := challenges.Issue(active.SubjectID, challenge.PurposePasswordReset, 10*time.Minute, 2)
	if err != nil {
		t.Fatalf("issue reset challenge: %v", err)
	}
	if err := challenges.Consume("usr_other-synthetic-subject", challenge.PurposePasswordReset, issuedChallenge.Token); !errors.Is(err, challenge.ErrInvalidChallenge) {
		t.Fatalf("cross-subject challenge should be rejected: %v", err)
	}
	if err := challenges.Consume(active.SubjectID, challenge.PurposePasswordReset, issuedChallenge.Token); err != nil {
		t.Fatalf("consume reset challenge: %v", err)
	}
	if err := challenges.Consume(active.SubjectID, challenge.PurposePasswordReset, issuedChallenge.Token); !errors.Is(err, challenge.ErrChallengeUsed) {
		t.Fatalf("reused reset challenge should be rejected: %v", err)
	}

	changed, err := identities.ChangePassword(ctx, active.SubjectID, active.AuthRevision, []byte("synthetic-password-2026"), []byte("synthetic-password-2027"))
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	if changed.AuthRevision == active.AuthRevision {
		t.Fatal("password change must advance authorization revision")
	}
	if _, err := sessions.Rotate(ctx, session.RotateInput{RefreshToken: issued.RefreshToken, ClientID: clientID, RedirectURI: redirectURI, Fingerprint: issueInput.Fingerprint}); !errors.Is(err, session.ErrSessionRevoked) {
		t.Fatalf("password change must revoke prior session family: %v", err)
	}
	if _, err := identities.Transition(ctx, changed.SubjectID, changed.AuthRevision, identity.AccountSuspended); err != nil {
		t.Fatalf("suspend synthetic identity: %v", err)
	}
}
