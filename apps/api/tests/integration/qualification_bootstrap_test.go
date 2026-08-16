package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/preproddata/canonicalaga"
	"github.com/aviason/aviaSurveil/internal/qualificationbootstrap"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestQualificationBootstrapReplayDriftAndPermissionBoundary(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "qualification_bootstrap")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	workspaceRoot := filepath.Clean(filepath.Join(apiModuleRoot(t), "..", "..", "..", ".."))
	foundationPath := filepath.Join(workspaceRoot, "deployments", "namibia", "manifests", "demo-foundation.json")
	rosterPath := filepath.Join(workspaceRoot, "deployments", "namibia", "manifests", "demo-identity-roster.json")
	catalogPath := filepath.Join(workspaceRoot, "deployments", "namibia", "manifests", "demo-approved-catalog.json")
	foundation, foundationDigest := readFoundation(t, foundationPath)
	roster, rosterDigest := readRoster(t, rosterPath)
	catalog := readCatalog(t, catalogPath)

	actor := "bootstrap:namibia/demo"
	now := time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)
	if err := qualificationbootstrap.LoadFoundation(ctx, pool, foundation, foundationDigest, actor, now); err != nil {
		t.Fatalf("load foundation: %v", err)
	}

	credentials := t.TempDir()
	for _, account := range roster.Accounts {
		if err := os.WriteFile(filepath.Join(credentials, account.PurposeToken), []byte("integration-test-credential"), 0o600); err != nil {
			t.Fatalf("write test credential: %v", err)
		}
	}
	provider := newQualificationProvider()
	if err := qualificationbootstrap.LoadRoster(ctx, pool, provider, roster, rosterDigest, "namibia/demo", "avia:first-party", credentials, actor, now); err != nil {
		t.Fatalf("load roster: %v", err)
	}

	packagePath := filepath.Join(apiModuleRoot(t), "..", "..", "deliverables", "AGA_ALL_FORMS_APPROVED_SOURCE_V2.zip")
	pkg, err := canonicalaga.ReadApprovedSourcePackage(ctx, packagePath, canonicalaga.ExactApprovedSourcePackage())
	if err != nil {
		t.Fatalf("read approved source package: %v", err)
	}
	if _, err := canonicalaga.LoadApprovedCatalog(ctx, pool, pkg, catalog.CatalogVersion, actor, catalog.ProviderScopeID, catalog.RegulatedTargetID, catalog.AdvisoryLockKey, now); err != nil {
		t.Fatalf("load approved catalog: %v", err)
	}

	beforeReplay := qualificationBootstrapCounts(t, pool)
	if err := qualificationbootstrap.LoadFoundation(ctx, pool, foundation, foundationDigest, actor, now.Add(time.Minute)); err != nil {
		t.Fatalf("replay foundation: %v", err)
	}
	if err := qualificationbootstrap.LoadRoster(ctx, pool, provider, roster, rosterDigest, "namibia/demo", "avia:first-party", credentials, actor, now.Add(time.Minute)); err != nil {
		t.Fatalf("replay roster: %v", err)
	}
	if _, err := canonicalaga.LoadApprovedCatalog(ctx, pool, pkg, catalog.CatalogVersion, actor, catalog.ProviderScopeID, catalog.RegulatedTargetID, catalog.AdvisoryLockKey, now.Add(time.Minute)); err != nil {
		t.Fatalf("replay approved catalog: %v", err)
	}
	afterReplay := qualificationBootstrapCounts(t, pool)
	if beforeReplay != afterReplay {
		t.Fatalf("bootstrap replay changed persisted counts: before=%v after=%v", beforeReplay, afterReplay)
	}
	if provider.provisionCalls != len(roster.Accounts) || provider.activationCalls != len(roster.Accounts) || provider.verifyCalls != 2*len(roster.Accounts) {
		t.Fatalf("provider reconciliation calls = provision %d activation %d verify %d, want provision/activation %d and verify %d", provider.provisionCalls, provider.activationCalls, provider.verifyCalls, len(roster.Accounts), 2*len(roster.Accounts))
	}

	managerSubjectID := provider.users[roster.Accounts[1].Email].SubjectID
	if _, err := pool.Exec(ctx, `INSERT INTO caa_departments (id, name, status) VALUES ('QUALIFICATION-EXTRA-DEPARTMENT', 'Qualification extra department', 'ACTIVE')`); err != nil {
		t.Fatalf("insert undeclared department: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO caa_organizational_units (id, department_id, name, status) VALUES ('QUALIFICATION-EXTRA-UNIT', 'QUALIFICATION-EXTRA-DEPARTMENT', 'Qualification extra unit', 'ACTIVE')`); err != nil {
		t.Fatalf("insert undeclared organizational unit: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO caa_department_memberships (id, root_id, subject_id, department_id, organizational_unit_id, membership_role, effective_from, status)
		VALUES ('qualification-extra-membership', 'qualification-extra-membership', $1, 'QUALIFICATION-EXTRA-DEPARTMENT', 'QUALIFICATION-EXTRA-UNIT', 'DEPARTMENT_MANAGER', '2026-08-15', 'ACTIVE')
	`, managerSubjectID); err != nil {
		t.Fatalf("insert undeclared active department authority: %v", err)
	}
	beforeUndeclaredAuthorityReplay := qualificationBootstrapCounts(t, pool)
	if err := qualificationbootstrap.LoadRoster(ctx, pool, provider, roster, rosterDigest, "namibia/demo", "avia:first-party", credentials, actor, now.Add(90*time.Second)); err == nil {
		t.Fatal("roster replay accepted an undeclared active department authority")
	}
	if got := qualificationBootstrapCounts(t, pool); got != beforeUndeclaredAuthorityReplay {
		t.Fatalf("undeclared department authority replay changed persisted counts: before=%v after=%v", beforeUndeclaredAuthorityReplay, got)
	}

	provider.driftEmail = roster.Accounts[0].Email
	provider.users[provider.driftEmail] = mutateProviderRole(provider.users[provider.driftEmail], "inspector")
	if err := qualificationbootstrap.LoadRoster(ctx, pool, provider, roster, rosterDigest, "namibia/demo", "avia:first-party", credentials, actor, now.Add(2*time.Minute)); err == nil {
		t.Fatal("roster drift was accepted")
	}
	if got := qualificationBootstrapCounts(t, pool); got != beforeUndeclaredAuthorityReplay {
		t.Fatalf("roster drift changed persisted counts: before=%v after=%v", beforeUndeclaredAuthorityReplay, got)
	}

	assertQualificationBootstrapPermissionBoundary(t, pool, foundation, foundationDigest, actor, now)
	assertSurveilBootstrapRoleBoundary(t, pool)
}

type qualificationCatalogManifest struct {
	CatalogVersion    string `json:"catalogVersion"`
	ProviderScopeID   string `json:"providerScopeId"`
	RegulatedTargetID string `json:"regulatedTargetId"`
	AdvisoryLockKey   int64  `json:"advisoryLockKey"`
}

func readFoundation(t *testing.T, path string) (qualificationbootstrap.FoundationManifest, string) {
	t.Helper()
	digest := fileSHA256(t, path)
	manifest, gotDigest, err := qualificationbootstrap.ReadFoundationManifest(path, digest, "namibia/demo")
	if err != nil {
		t.Fatalf("read foundation manifest: %v", err)
	}
	return manifest, gotDigest
}

func readRoster(t *testing.T, path string) (qualificationbootstrap.RosterManifest, string) {
	t.Helper()
	digest := fileSHA256(t, path)
	manifest, gotDigest, err := qualificationbootstrap.ReadRosterManifest(path, digest, "namibia/demo")
	if err != nil {
		t.Fatalf("read roster manifest: %v", err)
	}
	return manifest, gotDigest
}

func readCatalog(t *testing.T, path string) qualificationCatalogManifest {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read catalog manifest: %v", err)
	}
	var manifest qualificationCatalogManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("decode catalog manifest: %v", err)
	}
	if manifest.CatalogVersion == "" || manifest.ProviderScopeID == "" || manifest.RegulatedTargetID == "" {
		t.Fatal("catalog manifest is incomplete")
	}
	return manifest
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	digest := sha256.Sum256(contents)
	return "sha256:" + hex.EncodeToString(digest[:])
}

type qualificationBootstrapCount struct {
	Organizations             int
	RegulatedTargets          int
	ProviderScopes            int
	ProviderScopeTargets      int
	IdentityReferences        int
	UserProfiles              int
	UserSettings              int
	LifecycleRequests         int
	DesiredMembershipVersions int
	DesiredMembershipSync     int
	DepartmentMemberships     int
	Catalogs                  int
	CatalogForms              int
	QuestionVersions          int
	CatalogProvenance         int
	CatalogMemberships        int
	CatalogMembershipEvents   int
	CatalogApplicabilities    int
	CatalogImportRuns         int
}

func qualificationBootstrapCounts(t *testing.T, pool *database.Pool) qualificationBootstrapCount {
	t.Helper()
	var counts qualificationBootstrapCount
	err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM organizations),
			(SELECT count(*) FROM regulated_targets),
			(SELECT count(*) FROM organization_service_provider_scopes),
			(SELECT count(*) FROM organization_service_provider_scope_targets),
			(SELECT count(*) FROM identity_references),
			(SELECT count(*) FROM user_profiles),
			(SELECT count(*) FROM user_settings),
			(SELECT count(*) FROM user_lifecycle_requests),
			(SELECT count(*) FROM desired_membership_versions),
			(SELECT count(*) FROM desired_membership_sync),
			(SELECT count(*) FROM caa_department_memberships),
			(SELECT count(*) FROM canonical_question_catalogs),
			(SELECT count(*) FROM canonical_question_catalog_forms),
			(SELECT count(*) FROM question_versions),
			(SELECT count(*) FROM canonical_question_version_provenance),
			(SELECT count(*) FROM canonical_question_catalog_memberships),
			(SELECT count(*) FROM canonical_question_catalog_membership_events),
			(SELECT count(*) FROM canonical_question_catalog_applicabilities),
			(SELECT count(*) FROM canonical_question_catalog_import_runs)
	`).Scan(
		&counts.Organizations, &counts.RegulatedTargets, &counts.ProviderScopes, &counts.ProviderScopeTargets,
		&counts.IdentityReferences, &counts.UserProfiles, &counts.UserSettings, &counts.LifecycleRequests,
		&counts.DesiredMembershipVersions, &counts.DesiredMembershipSync, &counts.DepartmentMemberships,
		&counts.Catalogs, &counts.CatalogForms, &counts.QuestionVersions, &counts.CatalogProvenance,
		&counts.CatalogMemberships, &counts.CatalogMembershipEvents, &counts.CatalogApplicabilities,
		&counts.CatalogImportRuns,
	)
	if err != nil {
		t.Fatalf("read qualification bootstrap counts: %v", err)
	}
	return counts
}

func assertQualificationBootstrapPermissionBoundary(t *testing.T, pool *database.Pool, manifest qualificationbootstrap.FoundationManifest, manifestDigest, actor string, now time.Time) {
	t.Helper()
	baseURL := os.Getenv("AVIA_TEST_DATABASE_URL")
	if baseURL == "" {
		baseURL = "postgres://aviasurveil:aviasurveil@127.0.0.1:55432/aviasurveil?sslmode=disable"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse test database URL for permission test: %v", err)
	}
	var databaseName string
	if err := pool.QueryRow(context.Background(), "SELECT current_database()").Scan(&databaseName); err != nil {
		t.Fatalf("read test database name: %v", err)
	}
	role := fmt.Sprintf("qualification_readonly_%d", time.Now().UnixNano())
	password := "qualification-readonly-integration-password"
	if _, err := pool.Exec(context.Background(), fmt.Sprintf(`CREATE ROLE "%s" LOGIN PASSWORD '%s'`, role, password)); err != nil {
		t.Fatalf("create restricted integration role: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, role))
	})
	if _, err := pool.Exec(context.Background(), fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, databaseName, role)); err != nil {
		t.Fatalf("grant database connect: %v", err)
	}
	if _, err := pool.Exec(context.Background(), fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO "%s"`, role)); err != nil {
		t.Fatalf("grant schema usage: %v", err)
	}
	if _, err := pool.Exec(context.Background(), fmt.Sprintf(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO "%s"`, role)); err != nil {
		t.Fatalf("grant table read permission: %v", err)
	}

	restrictedURL := *parsed
	restrictedURL.Path = "/" + databaseName
	restrictedURL.User = url.UserPassword(role, password)
	restricted, err := database.Open(context.Background(), restrictedURL.String())
	if err != nil {
		t.Fatalf("open restricted integration pool: %v", err)
	}
	defer restricted.Close()
	if err := qualificationbootstrap.LoadFoundation(context.Background(), restricted, manifest, manifestDigest, actor, now); err == nil {
		t.Fatal("restricted database role was able to run foundation bootstrap")
	}
}

func assertSurveilBootstrapRoleBoundary(t *testing.T, pool *database.Pool) {
	t.Helper()
	ctx := context.Background()
	baseURL := os.Getenv("AVIA_TEST_DATABASE_URL")
	if baseURL == "" {
		baseURL = "postgres://aviasurveil:aviasurveil@127.0.0.1:55432/aviasurveil?sslmode=disable"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse bootstrap role database URL: %v", err)
	}
	if _, err := pool.Exec(ctx, `DROP ROLE IF EXISTS surveil_bootstrap`); err != nil {
		t.Fatalf("clear integration bootstrap role: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE ROLE surveil_bootstrap LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS PASSWORD 'bootstrap-role-integration-credential'`); err != nil {
		t.Fatalf("create integration bootstrap role: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DROP ROLE IF EXISTS surveil_bootstrap`) })
	var databaseName string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		t.Fatalf("read bootstrap role database name: %v", err)
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf(`GRANT CONNECT ON DATABASE %q TO surveil_bootstrap`, databaseName)); err != nil {
		t.Fatalf("grant bootstrap role database connect: %v", err)
	}
	if _, err := pool.Exec(ctx, `GRANT USAGE ON SCHEMA public TO surveil_bootstrap`); err != nil {
		t.Fatalf("grant bootstrap role schema usage: %v", err)
	}
	if _, err := pool.Exec(ctx, `REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC, surveil_bootstrap`); err != nil {
		t.Fatalf("revoke bootstrap role application function privileges: %v", err)
	}
	for _, table := range []string{
		"identity_references", "organizations", "regulated_targets",
		"organization_service_provider_scopes", "organization_service_provider_scope_targets",
		"service_provider_types", "user_profiles", "user_settings", "user_lifecycle_requests",
		"desired_membership_versions", "desired_membership_sync", "caa_department_memberships",
		"canonical_question_catalogs", "canonical_question_catalog_forms", "question_versions",
		"canonical_question_version_provenance", "canonical_question_catalog_memberships",
		"canonical_question_catalog_membership_events", "canonical_question_catalog_applicabilities",
		"canonical_question_catalog_import_runs",
	} {
		if _, err := pool.Exec(ctx, "GRANT SELECT, INSERT ON TABLE public."+table+" TO surveil_bootstrap"); err != nil {
			t.Fatalf("grant bootstrap role table %s: %v", table, err)
		}
	}
	restrictedURL := *parsed
	restrictedURL.Path = "/" + databaseName
	restrictedURL.User = url.UserPassword("surveil_bootstrap", "bootstrap-role-integration-credential")
	restricted, err := database.Open(ctx, restrictedURL.String())
	if err != nil {
		t.Fatalf("open real surveil_bootstrap role: %v", err)
	}
	defer restricted.Close()
	var count int
	if err := restricted.QueryRow(ctx, `SELECT count(*) FROM organizations`).Scan(&count); err != nil {
		t.Fatalf("surveil_bootstrap could not read an allowed foundation table: %v", err)
	}
	tx, err := restricted.Begin(ctx)
	if err != nil {
		t.Fatalf("begin bootstrap role insert check: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO identity_references(subject_id,issuer,display_name) VALUES ('bootstrap-permission-boundary','avia:test','permission boundary')`); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("surveil_bootstrap could not insert an allowed loader row: %v", err)
	}
	_ = tx.Rollback(ctx)
	if _, err := restricted.Exec(ctx, `UPDATE organizations SET legal_name = legal_name`); err == nil {
		t.Fatal("surveil_bootstrap can update unrelated or immutable data")
	}
	if _, err := restricted.Exec(ctx, `CREATE TABLE bootstrap_permission_boundary_forbidden(id text)`); err == nil {
		t.Fatal("surveil_bootstrap can create schema objects")
	}
	var canExecute bool
	if err := restricted.QueryRow(ctx, `SELECT has_function_privilege(current_user, 'public.reject_immutable_row_change()', 'EXECUTE')`).Scan(&canExecute); err != nil {
		t.Fatalf("inspect bootstrap function privilege: %v", err)
	}
	if canExecute {
		t.Fatal("surveil_bootstrap can execute application functions")
	}
}

type qualificationProvider struct {
	users           map[string]identity.ProviderDirectoryUser
	provisionCalls  int
	activationCalls int
	verifyCalls     int
	driftEmail      string
}

func newQualificationProvider() *qualificationProvider {
	return &qualificationProvider{users: make(map[string]identity.ProviderDirectoryUser)}
}

func (p *qualificationProvider) ListDirectory(_ context.Context, query identity.ProviderDirectoryQuery) (identity.ProviderDirectoryPage, error) {
	users := make([]identity.ProviderDirectoryUser, 0, len(p.users))
	for _, user := range p.users {
		if query.Search != "" && !strings.Contains(strings.ToLower(user.Email), strings.ToLower(query.Search)) {
			continue
		}
		users = append(users, user)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Email < users[j].Email })
	if query.First >= len(users) {
		return identity.ProviderDirectoryPage{}, nil
	}
	end := query.First + query.Limit
	if end > len(users) {
		end = len(users)
	}
	next := 0
	if end < len(users) {
		next = end
	}
	return identity.ProviderDirectoryPage{Users: users[query.First:end], NextFirst: next}, nil
}

func (p *qualificationProvider) ObserveUserAuthority(context.Context, string) (identity.AuthorityObservation, error) {
	return identity.AuthorityObservation{}, errors.New("not used by qualification bootstrap")
}
func (p *qualificationProvider) ProvisionUser(context.Context, identity.ProviderUser) (string, error) {
	return "", errors.New("unexpected non-revisioned provisioning")
}
func (p *qualificationProvider) ReconcileProvisionedUser(context.Context, identity.ProviderUser) (string, bool, error) {
	return "", false, errors.New("not used by qualification bootstrap")
}
func (p *qualificationProvider) DisableUser(context.Context, string) error {
	return errors.New("not used")
}
func (p *qualificationProvider) UpdateUserAuthority(context.Context, string, string, []identity.Role) error {
	return errors.New("not used")
}
func (p *qualificationProvider) EnableUser(context.Context, string) error {
	return errors.New("not used")
}
func (p *qualificationProvider) IssueExecuteActionsEmail(context.Context, string, []string, int) error {
	return errors.New("not used")
}
func (p *qualificationProvider) ResetUserMFA(context.Context, string) error {
	return errors.New("not used")
}
func (p *qualificationProvider) ForceUserLogout(context.Context, string) error {
	return errors.New("not used")
}
func (p *qualificationProvider) UpdateUserAuthorityAtRevision(context.Context, string, string, []identity.Role, string, int64, int64) error {
	return errors.New("not used")
}
func (p *qualificationProvider) SetUserStateAtRevision(context.Context, string, string, int64, int64) error {
	return errors.New("not used")
}

func (p *qualificationProvider) ProvisionUserAtRevision(_ context.Context, user identity.ProviderUser, expected, resulting int64) (string, error) {
	if expected != 0 || resulting != 1 {
		return "", errors.New("unexpected provisioning revision")
	}
	p.provisionCalls++
	subjectID := "provider-subject-" + fmt.Sprintf("%02d", p.provisionCalls)
	p.users[user.Email] = identity.ProviderDirectoryUser{
		SubjectID: subjectID, Email: user.Email, DisplayName: user.DisplayName,
		OrganizationID: user.OrganizationID, Roles: append([]identity.Role(nil), user.Roles...),
		MembershipID: user.MembershipID, MembershipRevision: 1, State: "INVITED",
	}
	return subjectID, nil
}

func (p *qualificationProvider) ActivateUserAtAuthorityRevision(_ context.Context, subjectID, membershipID string, expectedMembershipRevision, resultingMembershipRevision int64, expectedAuthRevision, resultingAuthRevision uint64, password string) error {
	if password == "" || expectedMembershipRevision != 1 || resultingMembershipRevision != 1 || expectedAuthRevision != 0 || resultingAuthRevision != 1 {
		return errors.New("unexpected activation revision")
	}
	for email, user := range p.users {
		if user.SubjectID == subjectID {
			if user.MembershipID != membershipID {
				return errors.New("activation membership mismatch")
			}
			user.State = "ACTIVE"
			user.Enabled = true
			user.AuthRevision = resultingAuthRevision
			p.users[email] = user
			p.activationCalls++
			return nil
		}
	}
	return errors.New("activation subject not found")
}

func (p *qualificationProvider) VerifyUserCredential(_ context.Context, subjectID, password string) (bool, error) {
	p.verifyCalls++
	for _, user := range p.users {
		if user.SubjectID == subjectID {
			return user.State == "ACTIVE" && user.Enabled && password != "", nil
		}
	}
	return false, errors.New("credential subject not found")
}

func mutateProviderRole(user identity.ProviderDirectoryUser, role string) identity.ProviderDirectoryUser {
	user.Roles = []identity.Role{identity.Role(role)}
	return user
}
