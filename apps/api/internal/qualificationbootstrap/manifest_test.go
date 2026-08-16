package qualificationbootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
)

const testTarget = "namibia/demo"

func TestReadRosterManifestAcceptsExactPreparedShape(t *testing.T) {
	manifest := RosterManifest{
		SchemaVersion:     1,
		ManifestVersion:   "test-roster-v1",
		AdvisoryLockKey:   41010202,
		Target:            testTarget,
		Enabled:           true,
		QualificationOnly: true,
		OnboardingMode:    "createAndDirectActivate",
		CredentialCustody: "target-secret-version",
		Accounts: []RosterAccount{
			{PurposeToken: "PLATFORM-ADMIN", DisplayName: "Admin", Email: "admin@example.test", OrganizationID: "CAA", Role: "admin", MembershipID: "membership-admin"},
			{PurposeToken: "AGA-MANAGER", DisplayName: "Manager", Email: "manager@example.test", OrganizationID: "CAA", Role: "manager", MembershipID: "membership-manager", Department: &DepartmentBinding{ID: "department-membership-manager", DepartmentID: "AERODROME_INSPECTORATE", OrganizationalUnitID: "AERODROME_INSPECTORATE"}},
			{PurposeToken: "FINANCE-REVIEWER", DisplayName: "Finance", Email: "finance@example.test", OrganizationID: "CAA", Role: "finance", MembershipID: "membership-finance"},
			{PurposeToken: "GENERAL-MANAGER", DisplayName: "GM", Email: "gm@example.test", OrganizationID: "CAA", Role: "gm", MembershipID: "membership-gm"},
			{PurposeToken: "EXECUTIVE-DIRECTOR", DisplayName: "ED", Email: "ed@example.test", OrganizationID: "CAA", Role: "executiveDirector", MembershipID: "membership-ed"},
			{PurposeToken: "LEAD-INSPECTOR", DisplayName: "Lead", Email: "lead@example.test", OrganizationID: "CAA", Role: "leadInspector", MembershipID: "membership-lead"},
			{PurposeToken: "INSPECTOR", DisplayName: "Inspector", Email: "inspector@example.test", OrganizationID: "CAA", Role: "inspector", MembershipID: "membership-inspector"},
			{PurposeToken: "TARGET-AUDITEE", DisplayName: "Target", Email: "target@example.test", OrganizationID: "ORG-TARGET", Role: "auditee", MembershipID: "membership-target"},
			{PurposeToken: "CONTROL-AUDITEE", DisplayName: "Control", Email: "control@example.test", OrganizationID: "ORG-CONTROL", Role: "auditee", MembershipID: "membership-control"},
		},
	}

	path := writeJSONManifest(t, manifest)
	digest := fileDigest(t, path)
	got, gotDigest, err := ReadRosterManifest(path, digest, testTarget)
	if err != nil {
		t.Fatalf("ReadRosterManifest() error = %v", err)
	}
	if gotDigest != digest || len(got.Accounts) != 9 || got.Accounts[1].Department == nil {
		t.Fatalf("ReadRosterManifest() = digest %q, accounts %d, manager department %v", gotDigest, len(got.Accounts), got.Accounts[1].Department)
	}
}

func TestReadRosterManifestRejectsUnknownFieldsAndSecretShapedData(t *testing.T) {
	unknown := `{"schemaVersion":1,"manifestVersion":"test","target":"namibia/demo","enabled":true,"qualificationOnly":true,"onboardingMode":"verifyExisting","credentialCustody":"none","accounts":[],"unexpected":true}`
	assertManifestRejected(t, []byte(unknown), "unknown field")

	secretMarker := `{"schemaVersion":1,"manifestVersion":"test","target":"namibia/demo","enabled":false,"qualificationOnly":true,"onboardingMode":"verifyExisting","credentialCustody":"none","accounts":[],"secretBytes":"redacted"}`
	assertManifestRejected(t, []byte(secretMarker), "secret-shaped field")
}

func TestReadRosterManifestRequiresOneManagerWithDepartmentAuthority(t *testing.T) {
	base := RosterManifest{
		SchemaVersion: 1, ManifestVersion: "test", AdvisoryLockKey: 41010202,
		Target: testTarget, Enabled: true, QualificationOnly: true,
		OnboardingMode: "verifyExisting", CredentialCustody: "none",
		Accounts: []RosterAccount{
			{PurposeToken: "MANAGER-ONE", DisplayName: "Manager One", Email: "one@example.test", OrganizationID: "CAA", Role: "manager", MembershipID: "membership-one"},
			{PurposeToken: "MANAGER-TWO", DisplayName: "Manager Two", Email: "two@example.test", OrganizationID: "CAA", Role: "manager", MembershipID: "membership-two"},
		},
	}
	path := writeJSONManifest(t, base)
	if _, _, err := ReadRosterManifest(path, fileDigest(t, path), testTarget); err == nil {
		t.Fatal("manifest with two manager roles and no department authority was accepted")
	}

	base.Accounts[1].Department = &DepartmentBinding{ID: "department-membership-two", DepartmentID: "AERODROME_INSPECTORATE", OrganizationalUnitID: "AERODROME_INSPECTORATE"}
	path = writeJSONManifest(t, base)
	if _, _, err := ReadRosterManifest(path, fileDigest(t, path), testTarget); err == nil {
		t.Fatal("manifest with two manager roles and one department authority was accepted")
	}

	base.Accounts = base.Accounts[:1]
	base.Accounts[0].Department = &DepartmentBinding{ID: "department-membership-one", DepartmentID: "AERODROME_INSPECTORATE", OrganizationalUnitID: "AERODROME_INSPECTORATE"}
	path = writeJSONManifest(t, base)
	if _, _, err := ReadRosterManifest(path, fileDigest(t, path), testTarget); err != nil {
		t.Fatalf("manifest with one manager and one department authority rejected: %v", err)
	}
}

func TestRosterCredentialRequiresPrivateRegularFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "TEST-ACCOUNT")
	if err := os.WriteFile(path, []byte("credential-value"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := rosterCredential(directory, "TEST-ACCOUNT")
	if err != nil || got != "credential-value" {
		t.Fatalf("rosterCredential() = %q, %v", got, err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := rosterCredential(directory, "TEST-ACCOUNT"); err == nil {
		t.Fatal("rosterCredential() accepted a credential file readable by group/other")
	}
}

func TestListCompleteDirectoryTraversesBoundedPages(t *testing.T) {
	provider := &directoryProvider{pages: []identity.ProviderDirectoryPage{
		{Users: []identity.ProviderDirectoryUser{{SubjectID: "subject-1", Email: "one@example.test"}}, NextFirst: 1},
		{Users: []identity.ProviderDirectoryUser{{SubjectID: "subject-2", Email: "two@example.test"}}, NextFirst: 0},
	}}
	users, err := listCompleteDirectory(context.Background(), provider)
	if err != nil {
		t.Fatalf("listCompleteDirectory() error = %v", err)
	}
	if len(users) != 2 || provider.calls != 2 || provider.firsts[0] != 0 || provider.firsts[1] != 1 {
		t.Fatalf("directory traversal = users %d, calls %d, firsts %v", len(users), provider.calls, provider.firsts)
	}
}

func TestListCompleteDirectoryRejectsNonAdvancingPagination(t *testing.T) {
	provider := &directoryProvider{pages: []identity.ProviderDirectoryPage{{NextFirst: 1}, {NextFirst: 1}}}
	if _, err := listCompleteDirectory(context.Background(), provider); err == nil {
		t.Fatal("listCompleteDirectory() accepted non-advancing pagination")
	}
}

func assertManifestRejected(t *testing.T, data []byte, label string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadRosterManifest(path, fileDigest(t, path), testTarget); err == nil {
		t.Fatalf("ReadRosterManifest() accepted %s", label)
	}
}

func writeJSONManifest(t *testing.T, manifest RosterManifest) string {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func fileDigest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

type directoryProvider struct {
	pages  []identity.ProviderDirectoryPage
	calls  int
	firsts []int
}

func (p *directoryProvider) ListDirectory(_ context.Context, query identity.ProviderDirectoryQuery) (identity.ProviderDirectoryPage, error) {
	p.firsts = append(p.firsts, query.First)
	if p.calls >= len(p.pages) {
		return identity.ProviderDirectoryPage{}, errors.New("unexpected directory page request")
	}
	page := p.pages[p.calls]
	p.calls++
	return page, nil
}

func (p *directoryProvider) ObserveUserAuthority(context.Context, string) (identity.AuthorityObservation, error) {
	return identity.AuthorityObservation{}, errors.New("not used")
}
func (p *directoryProvider) ProvisionUser(context.Context, identity.ProviderUser) (string, error) {
	return "", errors.New("not used")
}
func (p *directoryProvider) ReconcileProvisionedUser(context.Context, identity.ProviderUser) (string, bool, error) {
	return "", false, errors.New("not used")
}
func (p *directoryProvider) DisableUser(context.Context, string) error { return errors.New("not used") }
func (p *directoryProvider) UpdateUserAuthority(context.Context, string, string, []identity.Role) error {
	return errors.New("not used")
}
func (p *directoryProvider) EnableUser(context.Context, string) error { return errors.New("not used") }
func (p *directoryProvider) IssueExecuteActionsEmail(context.Context, string, []string, int) error {
	return errors.New("not used")
}
func (p *directoryProvider) ResetUserMFA(context.Context, string) error {
	return errors.New("not used")
}
func (p *directoryProvider) ForceUserLogout(context.Context, string) error {
	return errors.New("not used")
}
func (p *directoryProvider) UpdateUserAuthorityAtRevision(context.Context, string, string, []identity.Role, string, int64, int64) error {
	return errors.New("not used")
}
func (p *directoryProvider) ProvisionUserAtRevision(context.Context, identity.ProviderUser, int64, int64) (string, error) {
	return "", errors.New("not used")
}
func (p *directoryProvider) SetUserStateAtRevision(context.Context, string, string, int64, int64) error {
	return errors.New("not used")
}
func (p *directoryProvider) ActivateUserAtAuthorityRevision(context.Context, string, string, int64, int64, uint64, uint64, string) error {
	return errors.New("not used")
}
func (p *directoryProvider) VerifyUserCredential(context.Context, string, string) (bool, error) {
	return false, errors.New("not used")
}
