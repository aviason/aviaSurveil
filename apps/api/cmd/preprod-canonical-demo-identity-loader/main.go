package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

const fixtureSchema = "canonical-preprod-demo-identities/v2"

var (
	scenarioIDPattern   = regexp.MustCompile(`^synthetic-[a-z0-9-]{1,118}$`)
	membershipIDPattern = regexp.MustCompile(`^CANONICAL-DEMO-MEMBERSHIP-[A-Z0-9-]{1,96}$`)
	expectedRoles       = map[string]int{
		"admin": 1, "auditee": 2, "executiveDirector": 1, "finance": 1,
		"gm": 1, "inspector": 1, "leadInspector": 1, "manager": 1,
	}
)

type identityFixture struct {
	SchemaVersion string        `json:"schemaVersion"`
	Users         []fixtureUser `json:"users"`
}

type fixtureUser struct {
	ScenarioID           string                `json:"scenarioId"`
	MembershipID         string                `json:"membershipId"`
	Email                string                `json:"email"`
	DisplayName          string                `json:"displayName"`
	OrganizationID       string                `json:"organizationId"`
	Role                 string                `json:"role"`
	DepartmentMembership *departmentMembership `json:"departmentMembership,omitempty"`
}

type departmentMembership struct {
	ID                   string `json:"id"`
	DepartmentID         string `json:"departmentId"`
	OrganizationalUnitID string `json:"organizationalUnitId"`
}

type provisionedUser struct {
	Fixture   fixtureUser
	SubjectID string
}

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		slog.Error("canonical preprod demo identity load failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, output io.Writer) error {
	fixturePath := strings.TrimSpace(os.Getenv("AVIA_CANONICAL_DEMO_IDENTITIES_FILE"))
	issuer, err := validateIssuer(os.Getenv("AVIA_PREPROD_OIDC_ISSUER_URL"))
	if err != nil {
		return err
	}
	fixture, err := readFixture(fixturePath)
	if err != nil {
		return err
	}
	ownerPassword, err := readSecret(requiredEnvironment("AVIA_DATABASE_PASSWORD_FILE"))
	if err != nil {
		return err
	}
	qualificationPassword, err := readSecret(requiredEnvironment("AVIA_DEMO_QUALIFICATION_PASSWORD_FILE"))
	if err != nil {
		return err
	}
	if len(qualificationPassword) < 24 {
		return fmt.Errorf("demo qualification password is below the reviewed minimum")
	}
	providerAdmin, err := identity.NewFirstPartyAdminClient(identity.FirstPartyAdminConfig{
		BaseURL:    requiredEnvironment("AVIA_AUTH_ADMIN_URL"),
		SecretFile: requiredEnvironment("AVIA_AUTH_ADMIN_SECRET_FILE"),
	})
	if err != nil {
		return err
	}
	revisioned, ok := any(providerAdmin).(identity.RevisionedProviderAdmin)
	if !ok {
		return fmt.Errorf("first-party provider does not expose revisioned administration")
	}
	pool, err := database.Open(ctx, databaseURL(ownerPassword))
	if err != nil {
		return fmt.Errorf("open canonical demo identity database: %w", err)
	}
	defer pool.Close()
	if err := preflightDatabase(ctx, pool, fixture); err != nil {
		return err
	}

	provisioned := make([]provisionedUser, 0, len(fixture.Users))
	for _, user := range fixture.Users {
		subjectID, err := ensureProviderUser(ctx, providerAdmin, revisioned, user, qualificationPassword)
		if err != nil {
			return fmt.Errorf("prepare %s: %w", user.Role, err)
		}
		provisioned = append(provisioned, provisionedUser{Fixture: user, SubjectID: subjectID})
	}
	if err := verifyProviderDirectory(ctx, providerAdmin, provisioned); err != nil {
		return err
	}
	if err := persistAccounts(ctx, pool, provisioned, issuer, time.Now().UTC()); err != nil {
		return err
	}
	if err := verifyDatabase(ctx, pool, provisioned, issuer); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Canonical disposable first-party demo identities verified: accounts=%d roleFamilies=%d\n", len(provisioned), len(expectedRoles))
	return err
}

func ensureProviderUser(
	ctx context.Context,
	providerAdmin identity.ProviderAdmin,
	revisioned identity.RevisionedProviderAdmin,
	fixture fixtureUser,
	password string,
) (string, error) {
	user, found, err := findProviderUser(ctx, providerAdmin, fixture.Email)
	if err != nil {
		return "", err
	}
	if !found {
		subjectID, provisionErr := revisioned.ProvisionUserAtRevision(ctx, identity.ProviderUser{
			Email:          fixture.Email,
			DisplayName:    fixture.DisplayName,
			MembershipID:   fixture.MembershipID,
			OrganizationID: fixture.OrganizationID,
			Roles:          []identity.Role{identity.Role(fixture.Role)},
		}, 0, 1)
		if provisionErr != nil {
			return "", provisionErr
		}
		user, found, err = findProviderUser(ctx, providerAdmin, fixture.Email)
		if err != nil {
			return "", err
		}
		if !found || user.SubjectID != subjectID {
			return "", fmt.Errorf("provider account disappeared or changed subject after provisioning")
		}
	}
	if err := validateProviderDirectoryUser(user, fixture); err != nil {
		return "", err
	}
	if user.State == "INVITED" {
		direct, ok := any(providerAdmin).(interface {
			ActivateUserAtRevision(context.Context, string, string, int64, int64) error
		})
		if !ok {
			return "", fmt.Errorf("first-party provider does not expose disposable direct activation")
		}
		if err := direct.ActivateUserAtRevision(ctx, user.SubjectID, password, user.MembershipRevision, user.MembershipRevision); err != nil {
			return "", fmt.Errorf("direct auth activation failed: %w", err)
		}
	}
	updated, found, err := findProviderUser(ctx, providerAdmin, fixture.Email)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("provider account disappeared after activation")
	}
	if err := validateProviderDirectoryUser(updated, fixture); err != nil {
		return "", err
	}
	if updated.State != "ACTIVE" || !updated.Enabled {
		return "", fmt.Errorf("provider account %s is not active and verified", updated.SubjectID)
	}
	return updated.SubjectID, nil
}

func findProviderUser(ctx context.Context, providerAdmin identity.ProviderAdmin, email string) (identity.ProviderDirectoryUser, bool, error) {
	page, err := providerAdmin.ListDirectory(ctx, identity.ProviderDirectoryQuery{First: 0, Limit: 25, Search: email})
	if err != nil {
		return identity.ProviderDirectoryUser{}, false, err
	}
	matches := make([]identity.ProviderDirectoryUser, 0, 1)
	for _, user := range page.Users {
		if strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(email)) {
			matches = append(matches, user)
		}
	}
	if len(matches) > 1 {
		return identity.ProviderDirectoryUser{}, false, fmt.Errorf("provider directory contains duplicate email %q", email)
	}
	if len(matches) == 0 {
		return identity.ProviderDirectoryUser{}, false, nil
	}
	return matches[0], true, nil
}

func validateProviderDirectoryUser(user identity.ProviderDirectoryUser, fixture fixtureUser) error {
	if !strings.HasPrefix(user.SubjectID, "usr_") || len(user.SubjectID) != len("usr_")+22 ||
		!strings.EqualFold(user.Email, fixture.Email) || user.DisplayName != fixture.DisplayName ||
		user.OrganizationID != fixture.OrganizationID || user.MembershipID != fixture.MembershipID ||
		user.MembershipRevision != 1 || len(user.Roles) != 1 || string(user.Roles[0]) != fixture.Role {
		return fmt.Errorf("provider authority does not match the reviewed fixture for %s", fixture.Email)
	}
	return nil
}

func verifyProviderDirectory(ctx context.Context, providerAdmin identity.ProviderAdmin, users []provisionedUser) error {
	page, err := providerAdmin.ListDirectory(ctx, identity.ProviderDirectoryQuery{First: 0, Limit: 25})
	if err != nil {
		return fmt.Errorf("verify first-party provider directory: %w", err)
	}
	if len(page.Users) != len(users) || page.NextFirst != 0 {
		return fmt.Errorf("first-party provider directory count mismatch: users=%d expected=%d next=%d", len(page.Users), len(users), page.NextFirst)
	}
	expected := make(map[string]fixtureUser, len(users))
	for _, user := range users {
		expected[user.Fixture.Email] = user.Fixture
	}
	seen := make(map[string]bool, len(users))
	for _, candidate := range page.Users {
		fixture, ok := expected[candidate.Email]
		if !ok || seen[candidate.Email] {
			return fmt.Errorf("first-party provider directory contains an unexpected or duplicate account")
		}
		if err := validateProviderDirectoryUser(candidate, fixture); err != nil {
			return fmt.Errorf("first-party provider directory contains a drifted account: %w", err)
		}
		if candidate.State != "ACTIVE" || !candidate.Enabled {
			return fmt.Errorf("first-party provider directory contains an inactive account")
		}
		seen[candidate.Email] = true
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("first-party provider directory omitted a fixture account")
	}
	return nil
}

func readFixture(path string) (identityFixture, error) {
	if !filepath.IsAbs(path) {
		return identityFixture{}, fmt.Errorf("demo identity fixture path must be absolute")
	}
	file, err := os.Open(path)
	if err != nil {
		return identityFixture{}, fmt.Errorf("open demo identity fixture: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64*1024))
	decoder.DisallowUnknownFields()
	var fixture identityFixture
	if err := decoder.Decode(&fixture); err != nil {
		return identityFixture{}, fmt.Errorf("decode demo identity fixture: %w", err)
	}
	if err := validateFixture(fixture); err != nil {
		return identityFixture{}, err
	}
	return fixture, nil
}

func validateFixture(fixture identityFixture) error {
	if fixture.SchemaVersion != fixtureSchema || len(fixture.Users) != 9 {
		return fmt.Errorf("demo identity fixture must contain the exact reviewed matrix")
	}
	roleCounts := make(map[string]int, len(expectedRoles))
	scenariosSeen := make(map[string]bool, len(fixture.Users))
	membershipsSeen := make(map[string]bool, len(fixture.Users))
	emailsSeen := make(map[string]bool, len(fixture.Users))
	managerDepartments := 0
	for _, user := range fixture.Users {
		address, addressErr := mail.ParseAddress(user.Email)
		if !scenarioIDPattern.MatchString(user.ScenarioID) ||
			!membershipIDPattern.MatchString(user.MembershipID) ||
			addressErr != nil || address.Address != user.Email ||
			!strings.HasSuffix(user.Email, "@synthetic.invalid") ||
			strings.TrimSpace(user.DisplayName) == "" ||
			(user.OrganizationID != "CAA" && user.OrganizationID != "ORG-FLY-NAMIBIA") ||
			scenariosSeen[user.ScenarioID] || membershipsSeen[user.MembershipID] || emailsSeen[user.Email] {
			return fmt.Errorf("demo identity fixture contains an invalid or duplicate user")
		}
		if _, ok := expectedRoles[user.Role]; !ok || (user.Role == "auditee") == (user.OrganizationID == "CAA") {
			return fmt.Errorf("demo identity role and organization differ from the reviewed matrix")
		}
		if user.DepartmentMembership != nil {
			managerDepartments++
			if user.Role != "manager" || user.DepartmentMembership.ID != "CANONICAL-DEMO-DEPARTMENT-MANAGER" ||
				user.DepartmentMembership.DepartmentID != "FLIGHT_OPERATIONS_INSPECTORATE" ||
				user.DepartmentMembership.OrganizationalUnitID != "FLIGHT_OPERATIONS_INSPECTORATE" {
				return fmt.Errorf("demo Department Manager authority differs from the reviewed boundary")
			}
		}
		scenariosSeen[user.ScenarioID] = true
		membershipsSeen[user.MembershipID] = true
		emailsSeen[user.Email] = true
		roleCounts[user.Role]++
	}
	if managerDepartments != 1 || len(roleCounts) != len(expectedRoles) {
		return fmt.Errorf("demo identity authority matrix is incomplete")
	}
	for role, count := range expectedRoles {
		if roleCounts[role] != count {
			return fmt.Errorf("demo identity role %s count differs", role)
		}
	}
	return nil
}

func validateIssuer(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		parsed.Path != "/identity" || parsed.String() != value {
		return "", fmt.Errorf("exact canonical local-preprod first-party OIDC issuer is required")
	}
	if parsed.Scheme == "http" && parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1" {
		return "", fmt.Errorf("HTTP canonical issuer is allowed only on loopback")
	}
	return value, nil
}

func preflightDatabase(ctx context.Context, pool *database.Pool, fixture identityFixture) error {
	expected := make(map[string]bool, len(fixture.Users))
	for _, user := range fixture.Users {
		expected[strings.ToLower(user.Email)] = true
	}
	rows, err := pool.Query(ctx, `SELECT email FROM identity_references WHERE email LIKE '%@synthetic.invalid' ORDER BY email`)
	if err != nil {
		return fmt.Errorf("preflight demo identity database: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return fmt.Errorf("scan demo identity preflight: %w", err)
		}
		if !expected[strings.ToLower(email)] {
			return fmt.Errorf("canonical demo identity database contains an unexpected synthetic account")
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate demo identity preflight: %w", err)
	}
	return nil
}

func persistAccounts(ctx context.Context, pool *database.Pool, users []provisionedUser, issuer string, now time.Time) error {
	if len(users) != 9 || now.IsZero() {
		return fmt.Errorf("exact provisioned identity matrix and timestamp are required")
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		for _, user := range users {
			if err := persistAccount(ctx, tx, user, issuer, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func persistAccount(ctx context.Context, tx pgx.Tx, user provisionedUser, issuer string, now time.Time) error {
	requestID := "CANONICAL-DEMO-PROVISION-" + user.SubjectID
	presence, err := applicationPresence(ctx, tx, user, requestID)
	if err != nil {
		return err
	}
	allPresent := presence.identity && presence.profile && presence.lifecycle && presence.version && presence.sync && presence.department == (user.Fixture.DepartmentMembership != nil)
	anyPresent := presence.identity || presence.profile || presence.lifecycle || presence.version || presence.sync || presence.department
	if allPresent {
		if err := validatePersistedAccount(ctx, tx, user, issuer, requestID); err != nil {
			return err
		}
		return nil
	}
	if anyPresent {
		return fmt.Errorf("application identity state is partial for provider subject %s", user.SubjectID)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO identity_references
			(subject_id, issuer, display_name, revision, email, created_at)
		VALUES ($1,$2,$3,1,$4,$5)`, user.SubjectID, issuer, user.Fixture.DisplayName, user.Fixture.Email, now); err != nil {
		return fmt.Errorf("insert demo identity reference: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_profiles
			(subject_id, display_name, organization_id, revision, created_at, updated_at)
		VALUES ($1,$2,$3,1,$4,$4)`, user.SubjectID, user.Fixture.DisplayName, user.Fixture.OrganizationID, now); err != nil {
		return fmt.Errorf("insert demo user profile: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_lifecycle_requests (
			id, subject_id, requested_action, requested_roles,
			requested_organization_id, status, idempotency_key,
			requested_by_subject_id, requested_email, requested_display_name,
			expected_membership_revision, reason, requested_effective_at,
			membership_id, resulting_membership_revision,
			provider_acknowledged_at, created_at, updated_at
		) VALUES ($1,$2,'PROVISION',$3,$4,'SUCCEEDED',$5,$2,$6,$7,0,$8,$9,$10,1,$9,$9,$9)
	`, requestID, user.SubjectID, []string{user.Fixture.Role}, user.Fixture.OrganizationID,
		"canonical-local-demo-provision:"+user.SubjectID, user.Fixture.Email, user.Fixture.DisplayName,
		"Disposable local canonical first-party demo account.", now, user.Fixture.MembershipID); err != nil {
		return fmt.Errorf("insert demo lifecycle receipt: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO desired_membership_versions (
			membership_id, subject_id, revision, membership_state,
			organization_id, roles, requested_by_subject_id, reason,
			source_request_id, requested_at, effective_at,
			observed_provider_enabled, observed_organization_id,
			observed_roles, observed_at, drift_state
		) VALUES ($1,$2,1,'ACTIVE',$3,$4,$2,$5,$6,$7,$7,true,$3,$4,$7,'IN_SYNC')
	`, user.Fixture.MembershipID, user.SubjectID, user.Fixture.OrganizationID,
		[]string{user.Fixture.Role}, "Disposable local canonical first-party demo account.", requestID, now); err != nil {
		return fmt.Errorf("insert demo membership version: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO desired_membership_sync (
			membership_id, subject_id, desired_revision,
			observed_provider_enabled, observed_organization_id,
			observed_roles, observed_at, drift_state
		) VALUES ($1,$2,1,true,$3,$4,$5,'IN_SYNC')
	`, user.Fixture.MembershipID, user.SubjectID, user.Fixture.OrganizationID,
		[]string{user.Fixture.Role}, now); err != nil {
		return fmt.Errorf("insert demo membership sync: %w", err)
	}
	if department := user.Fixture.DepartmentMembership; department != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO caa_department_memberships (
				id, subject_id, department_id, organizational_unit_id,
				membership_role, effective_from, status, created_at, root_id
			) VALUES ($1,$2,$3,$4,'DEPARTMENT_MANAGER',$5::date,'ACTIVE',$6,$1)
		`, department.ID, user.SubjectID, department.DepartmentID,
			department.OrganizationalUnitID, now.Format(time.DateOnly), now); err != nil {
			return fmt.Errorf("insert demo Department Manager authority: %w", err)
		}
	}
	return nil
}

type applicationPresenceState struct {
	identity, profile, lifecycle, version, sync, department bool
}

func applicationPresence(ctx context.Context, tx pgx.Tx, user provisionedUser, requestID string) (applicationPresenceState, error) {
	var state applicationPresenceState
	departmentID := ""
	if user.Fixture.DepartmentMembership != nil {
		departmentID = user.Fixture.DepartmentMembership.ID
	}
	if err := tx.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM identity_references WHERE subject_id = $1),
			EXISTS (SELECT 1 FROM user_profiles WHERE subject_id = $1),
			EXISTS (SELECT 1 FROM user_lifecycle_requests WHERE id = $2),
			EXISTS (SELECT 1 FROM desired_membership_versions WHERE membership_id = $3),
			EXISTS (SELECT 1 FROM desired_membership_sync WHERE membership_id = $3),
			($4 <> '' AND EXISTS (SELECT 1 FROM caa_department_memberships WHERE id = $4))
	`, user.SubjectID, requestID, user.Fixture.MembershipID, departmentID).Scan(
		&state.identity, &state.profile, &state.lifecycle, &state.version, &state.sync, &state.department); err != nil {
		return applicationPresenceState{}, fmt.Errorf("inspect application identity state: %w", err)
	}
	var existingSubject string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(subject_id, '') FROM identity_references WHERE lower(email) = lower($1)`, user.Fixture.Email).Scan(&existingSubject); err == nil && existingSubject != user.SubjectID {
		return applicationPresenceState{}, fmt.Errorf("application email is bound to a different provider subject")
	} else if err != nil && err != pgx.ErrNoRows {
		return applicationPresenceState{}, fmt.Errorf("inspect application email binding: %w", err)
	}
	return state, nil
}

func validatePersistedAccount(ctx context.Context, tx pgx.Tx, user provisionedUser, issuer, requestID string) error {
	var identityOK, profileOK, lifecycleOK, versionOK, syncOK, departmentOK bool
	departmentID := ""
	if user.Fixture.DepartmentMembership != nil {
		departmentID = user.Fixture.DepartmentMembership.ID
	}
	if err := tx.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM identity_references WHERE subject_id = $1 AND issuer = $2 AND display_name = $3 AND revision = 1 AND email = $4 AND tombstoned_at IS NULL),
			EXISTS (SELECT 1 FROM user_profiles WHERE subject_id = $1 AND display_name = $3 AND organization_id = $5 AND revision = 1 AND tombstoned_at IS NULL),
			EXISTS (SELECT 1 FROM user_lifecycle_requests WHERE id = $6 AND subject_id = $1 AND requested_action = 'PROVISION' AND requested_roles = $7 AND requested_organization_id = $5 AND status = 'SUCCEEDED' AND idempotency_key = $8 AND requested_by_subject_id = $1 AND requested_email = $4 AND requested_display_name = $3 AND expected_membership_revision = 0 AND membership_id = $9 AND resulting_membership_revision = 1),
			EXISTS (SELECT 1 FROM desired_membership_versions WHERE membership_id = $9 AND subject_id = $1 AND revision = 1 AND membership_state = 'ACTIVE' AND organization_id = $5 AND roles = $7 AND requested_by_subject_id = $1 AND source_request_id = $6 AND observed_provider_enabled = true AND observed_organization_id = $5 AND observed_roles = $7 AND drift_state = 'IN_SYNC'),
			EXISTS (SELECT 1 FROM desired_membership_sync WHERE membership_id = $9 AND subject_id = $1 AND desired_revision = 1 AND observed_provider_enabled = true AND observed_organization_id = $5 AND observed_roles = $7 AND drift_state = 'IN_SYNC'),
			($10 <> '' AND EXISTS (SELECT 1 FROM caa_department_memberships WHERE id = $10 AND subject_id = $1 AND department_id = 'FLIGHT_OPERATIONS_INSPECTORATE' AND organizational_unit_id = 'FLIGHT_OPERATIONS_INSPECTORATE' AND membership_role = 'DEPARTMENT_MANAGER' AND status = 'ACTIVE'))
	`, user.SubjectID, issuer, user.Fixture.DisplayName, user.Fixture.Email, user.Fixture.OrganizationID,
		requestID, []string{user.Fixture.Role}, "canonical-local-demo-provision:"+user.SubjectID, user.Fixture.MembershipID, departmentID).Scan(
		&identityOK, &profileOK, &lifecycleOK, &versionOK, &syncOK, &departmentOK); err != nil {
		return fmt.Errorf("validate persisted application identity state: %w", err)
	}
	if !identityOK || !profileOK || !lifecycleOK || !versionOK || !syncOK || departmentOK != (user.Fixture.DepartmentMembership != nil) {
		return fmt.Errorf("application identity state drifted for provider subject %s", user.SubjectID)
	}
	return nil
}

func verifyDatabase(ctx context.Context, pool *database.Pool, users []provisionedUser, issuer string) error {
	var identities, profiles, memberships, syncRows, managerAuthorities int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM identity_references WHERE email LIKE '%@synthetic.invalid'),
			(SELECT count(*) FROM user_profiles WHERE subject_id IN (SELECT subject_id FROM identity_references WHERE email LIKE '%@synthetic.invalid')),
			(SELECT count(*) FROM desired_membership_versions WHERE membership_id LIKE 'CANONICAL-DEMO-MEMBERSHIP-%' AND revision = 1 AND membership_state = 'ACTIVE'),
			(SELECT count(*) FROM desired_membership_sync WHERE membership_id LIKE 'CANONICAL-DEMO-MEMBERSHIP-%' AND desired_revision = 1 AND drift_state = 'IN_SYNC'),
			(SELECT count(*) FROM caa_department_memberships WHERE root_id = 'CANONICAL-DEMO-DEPARTMENT-MANAGER' AND status = 'ACTIVE')
	`).Scan(&identities, &profiles, &memberships, &syncRows, &managerAuthorities); err != nil {
		return fmt.Errorf("verify demo identity database: %w", err)
	}
	if identities != 9 || profiles != 9 || memberships != 9 || syncRows != 9 || managerAuthorities != 1 {
		return fmt.Errorf("demo identity seed count mismatch: identities=%d profiles=%d memberships=%d sync=%d managerAuthorities=%d", identities, profiles, memberships, syncRows, managerAuthorities)
	}
	for _, user := range users {
		var storedIssuer, storedSubject string
		if err := pool.QueryRow(ctx, `SELECT subject_id, issuer FROM identity_references WHERE email = $1`, user.Fixture.Email).Scan(&storedSubject, &storedIssuer); err != nil {
			return fmt.Errorf("verify application identity %s: %w", user.Fixture.Email, err)
		}
		if storedSubject != user.SubjectID || storedIssuer != issuer {
			return fmt.Errorf("application identity reference does not match first-party provider subject")
		}
	}
	return nil
}

func databaseURL(password string) string {
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword("aviasurveil360_preprod_loader", password),
		Host:     net.JoinHostPort("preprod-postgres", "5432"),
		Path:     "aviasurveil360_local_preprod",
		RawQuery: "sslmode=disable",
	}).String()
}

func requiredEnvironment(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}

func readSecret(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("runtime secret path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 1 || info.Size() > 4096 {
		return "", fmt.Errorf("runtime secret must be a bounded regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(data))
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("runtime secret is empty or malformed")
	}
	return value, nil
}
