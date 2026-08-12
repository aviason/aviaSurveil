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

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/preproddata/scenarios"
	"github.com/jackc/pgx/v5"
)

const (
	fixtureSchema = "canonical-preprod-demo-identities/v1"
	realmName     = "aviasurveil360-local-preprod"
)

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
	Fixture fixtureUser
	Account scenarios.ProviderAccount
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
	serviceSecret, err := readSecret(requiredEnvironment("AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET_FILE"))
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

	pool, err := database.Open(ctx, databaseURL(ownerPassword))
	if err != nil {
		return fmt.Errorf("open canonical demo identity database: %w", err)
	}
	defer pool.Close()
	if err := preflightDatabase(ctx, pool); err != nil {
		return err
	}

	keycloak, err := scenarios.NewKeycloakEndpoint(scenarios.KeycloakEndpointConfig{
		BaseURL:      requiredEnvironment("AVIA_KEYCLOAK_ADMIN_URL"),
		Realm:        realmName,
		ClientID:     requiredEnvironment("AVIA_KEYCLOAK_SERVICE_CLIENT_ID"),
		ClientSecret: serviceSecret,
	})
	if err != nil {
		return err
	}
	if err := keycloak.Preflight(ctx); err != nil {
		return fmt.Errorf("preflight canonical demo Keycloak target: %w", err)
	}

	provisioned := make([]provisionedUser, 0, len(fixture.Users))
	accounts := make([]scenarios.ProviderAccount, 0, len(fixture.Users))
	for _, user := range fixture.Users {
		account, err := keycloak.EnsureProviderAccount(ctx, scenarios.ProviderAccount{
			ScenarioID:      user.ScenarioID,
			MembershipID:    user.MembershipID,
			Email:           user.Email,
			OrganizationID:  user.OrganizationID,
			Role:            user.Role,
			Enabled:         true,
			RequiredActions: []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
		})
		if err != nil {
			return fmt.Errorf("provision %s: %w", user.Role, err)
		}
		accounts = append(accounts, account)
		provisioned = append(provisioned, provisionedUser{Fixture: user, Account: account})
	}
	if err := keycloak.QualifyExistingProviderAccounts(ctx, accounts, qualificationPassword); err != nil {
		return fmt.Errorf("qualify canonical demo Keycloak accounts: %w", err)
	}
	if err := persistAccounts(ctx, pool, provisioned, issuer, time.Now().UTC()); err != nil {
		return err
	}
	if err := verifyDatabase(ctx, pool); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Canonical disposable demo identities verified: accounts=%d roleFamilies=%d\n", len(accounts), len(expectedRoles))
	return err
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
		parsed.Path != "/identity/realms/"+realmName || parsed.String() != value {
		return "", fmt.Errorf("exact canonical local-preprod OIDC issuer is required")
	}
	return value, nil
}

func preflightDatabase(ctx context.Context, pool *database.Pool) error {
	var syntheticIdentities, demoMemberships int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM identity_references WHERE email LIKE '%@synthetic.invalid'),
			(SELECT count(*) FROM desired_membership_versions WHERE membership_id LIKE 'CANONICAL-DEMO-MEMBERSHIP-%')
	`).Scan(&syntheticIdentities, &demoMemberships); err != nil {
		return fmt.Errorf("preflight demo identity database: %w", err)
	}
	if syntheticIdentities != 0 || demoMemberships != 0 {
		return fmt.Errorf("canonical demo identity database is not empty")
	}
	return nil
}

func persistAccounts(ctx context.Context, pool *database.Pool, users []provisionedUser, issuer string, now time.Time) error {
	if len(users) != 9 || now.IsZero() {
		return fmt.Errorf("exact provisioned identity matrix and timestamp are required")
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		for _, user := range users {
			if _, err := tx.Exec(ctx, `
				INSERT INTO identity_references
					(subject_id, issuer, display_name, revision, email, created_at)
				VALUES ($1,$2,$3,1,$4,$5)
			`, user.Account.SubjectID, issuer, user.Fixture.DisplayName, user.Fixture.Email, now); err != nil {
				return fmt.Errorf("insert demo identity reference: %w", err)
			}
		}
		for _, user := range users {
			requestID := "CANONICAL-DEMO-PROVISION-" + user.Account.SubjectID
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_lifecycle_requests (
					id, subject_id, requested_action, requested_roles,
					requested_organization_id, status, idempotency_key,
					requested_by_subject_id, requested_email, requested_display_name,
					expected_membership_revision, reason, requested_effective_at,
					membership_id, resulting_membership_revision,
					provider_acknowledged_at, created_at, updated_at
				) VALUES ($1,$2,'PROVISION',$3,$4,'SUCCEEDED',$5,$2,$6,$7,0,$8,$9,$10,1,$9,$9,$9)
			`, requestID, user.Account.SubjectID, []string{user.Fixture.Role},
				user.Fixture.OrganizationID, "canonical-local-demo-provision:"+user.Account.SubjectID,
				user.Fixture.Email, user.Fixture.DisplayName,
				"Disposable local canonical AGA demo account.", now, user.Fixture.MembershipID); err != nil {
				return fmt.Errorf("insert demo lifecycle receipt: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_profiles
					(subject_id, display_name, organization_id, revision, created_at, updated_at)
				VALUES ($1,$2,$3,1,$4,$4)
			`, user.Account.SubjectID, user.Fixture.DisplayName, user.Fixture.OrganizationID, now); err != nil {
				return fmt.Errorf("insert demo user profile: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO desired_membership_versions (
					membership_id, subject_id, revision, membership_state,
					organization_id, roles, requested_by_subject_id, reason,
					source_request_id, requested_at, effective_at,
					observed_provider_enabled, observed_organization_id,
					observed_roles, observed_at, drift_state
				) VALUES ($1,$2,1,'ACTIVE',$3,$4,$2,$5,$6,$7,$7,true,$3,$4,$7,'IN_SYNC')
			`, user.Fixture.MembershipID, user.Account.SubjectID, user.Fixture.OrganizationID,
				[]string{user.Fixture.Role}, "Disposable local canonical AGA demo account.",
				requestID, now); err != nil {
				return fmt.Errorf("insert demo membership version: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO desired_membership_sync (
					membership_id, subject_id, desired_revision,
					observed_provider_enabled, observed_organization_id,
					observed_roles, observed_at, drift_state
				) VALUES ($1,$2,1,true,$3,$4,$5,'IN_SYNC')
			`, user.Fixture.MembershipID, user.Account.SubjectID,
				user.Fixture.OrganizationID, []string{user.Fixture.Role}, now); err != nil {
				return fmt.Errorf("insert demo membership sync: %w", err)
			}
			if department := user.Fixture.DepartmentMembership; department != nil {
				if _, err := tx.Exec(ctx, `
					INSERT INTO caa_department_memberships (
						id, subject_id, department_id, organizational_unit_id,
						membership_role, effective_from, status, created_at, root_id
					) VALUES ($1,$2,$3,$4,'DEPARTMENT_MANAGER',$5::date,'ACTIVE',$6,$1)
				`, department.ID, user.Account.SubjectID, department.DepartmentID,
					department.OrganizationalUnitID, now.Format(time.DateOnly), now); err != nil {
					return fmt.Errorf("insert demo Department Manager authority: %w", err)
				}
			}
		}
		return nil
	})
}

func verifyDatabase(ctx context.Context, pool *database.Pool) error {
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
