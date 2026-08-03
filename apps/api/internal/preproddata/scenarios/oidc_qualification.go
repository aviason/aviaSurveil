package scenarios

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

var qualificationRoleCounts = map[string]int{
	"inspector": 1, "leadInspector": 1, "manager": 1, "finance": 1,
	"gm": 1, "executiveDirector": 1, "auditee": 2, "admin": 1,
}

// QualificationAccounts converts only the exact frozen smoke-profile provider
// rows. It refuses to synthesize, add, or infer an identity.
func QualificationAccounts(records []Record) ([]ProviderAccount, error) {
	if len(records) != 9 {
		return nil, fmt.Errorf("OIDC qualification requires exactly nine existing provider accounts")
	}
	accounts := make([]ProviderAccount, 0, len(records))
	roles := make(map[string]int, len(qualificationRoleCounts))
	seenSubjects := make(map[string]bool, len(records))
	seenMemberships := make(map[string]bool, len(records))
	for _, record := range records {
		account, err := providerAccountFromRecord(record)
		if err != nil {
			return nil, err
		}
		if seenSubjects[account.SubjectID] || seenMemberships[account.MembershipID] {
			return nil, fmt.Errorf("OIDC qualification account identity is duplicated")
		}
		seenSubjects[account.SubjectID] = true
		seenMemberships[account.MembershipID] = true
		roles[account.Role]++
		accounts = append(accounts, account)
	}
	for role, count := range qualificationRoleCounts {
		if roles[role] != count {
			return nil, fmt.Errorf("OIDC qualification role matrix differs from frozen smoke profile")
		}
	}
	if len(roles) != len(qualificationRoleCounts) {
		return nil, fmt.Errorf("OIDC qualification contains an unexpected role")
	}
	sort.Slice(accounts, func(left, right int) bool {
		return accounts[left].Role < accounts[right].Role ||
			(accounts[left].Role == accounts[right].Role && accounts[left].Email < accounts[right].Email)
	})
	return accounts, nil
}

// ActivateQualificationAccounts appends an ACTIVE predecessor revision for
// each exact synthetic smoke account using that account's provider-bound
// authority. The frozen smoke lifecycle intentionally exercises role and
// organization drift, so its revision-two authority is validated as a real
// predecessor but is not treated as the provider-account authority. This is
// deliberately separate from the AGA loader and must run before the overlay
// operation snapshots are taken.
func ActivateQualificationAccounts(
	ctx context.Context,
	pool *database.Pool,
	accounts []ProviderAccount,
	issuer string,
	now time.Time,
) error {
	if pool == nil || strings.TrimSpace(issuer) == "" || now.IsZero() {
		return fmt.Errorf("OIDC qualification database target, issuer, and time are required")
	}
	if len(accounts) != 9 {
		return fmt.Errorf("OIDC qualification requires the exact account matrix")
	}
	now = now.UTC()
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		for _, account := range accounts {
			var revision int64
			var subjectID, organizationID, state string
			var roles []string
			if err := tx.QueryRow(ctx, `
				SELECT revision, subject_id, organization_id, roles, membership_state
				FROM desired_membership_versions
				WHERE membership_id = $1
				ORDER BY revision DESC LIMIT 1
			`, account.MembershipID).Scan(&revision, &subjectID, &organizationID, &roles, &state); err != nil {
				return fmt.Errorf("read OIDC qualification membership: %w", err)
			}
			qualificationOrganizationID, qualificationRoles, err :=
				qualificationAuthorityFor(
					account,
					revision,
					subjectID,
					organizationID,
					roles,
					state,
				)
			if err != nil {
				return err
			}
			requestID := "aga-demo-oidc-qualification-" + account.MembershipID
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_lifecycle_requests (
					id, subject_id, requested_action, requested_roles,
					requested_organization_id, status, idempotency_key,
					requested_by_subject_id, expected_membership_revision,
					reason, requested_effective_at, membership_id,
					resulting_membership_revision, provider_acknowledged_at,
					created_at, updated_at
				) VALUES ($1,$2,'UPDATE_ROLES',$3,$4,'SUCCEEDED',$5,$2,2,$6,$7,$8,3,$7,$7,$7)
			`, requestID, account.SubjectID, qualificationRoles, qualificationOrganizationID,
				"preprod-aga-demo-oidc-qualification:"+account.MembershipID,
				"Separately authorized disposable OIDC qualification fixture.", now,
				account.MembershipID); err != nil {
				return fmt.Errorf("append OIDC qualification request: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO desired_membership_versions (
					membership_id, subject_id, revision, membership_state,
					organization_id, roles, requested_by_subject_id, reason,
					source_request_id, requested_at, effective_at,
					observed_provider_enabled, observed_organization_id,
					observed_roles, observed_at, drift_state
				) VALUES ($1,$2,3,'ACTIVE',$3,$4,$2,$5,$6,$7,$7,true,$3,$4,$7,'IN_SYNC')
			`, account.MembershipID, account.SubjectID, qualificationOrganizationID,
				qualificationRoles, "Separately authorized disposable OIDC qualification fixture.",
				requestID, now); err != nil {
				return fmt.Errorf("append active OIDC qualification membership: %w", err)
			}
			result, err := tx.Exec(ctx, `
				UPDATE desired_membership_sync
				SET desired_revision = 3, observed_provider_enabled = true,
				    observed_organization_id = $2, observed_roles = $3,
				    observed_at = $4, drift_state = 'IN_SYNC'
				WHERE membership_id = $1 AND subject_id = $5 AND desired_revision = 2
			`, account.MembershipID, qualificationOrganizationID, qualificationRoles, now, account.SubjectID)
			if err != nil || result.RowsAffected() != 1 {
				return fmt.Errorf("advance OIDC qualification sync: rows=%d error=%v", result.RowsAffected(), err)
			}
			result, err = tx.Exec(ctx, `
				UPDATE user_profiles
				SET organization_id = $2, revision = revision + 1, updated_at = $3
				WHERE subject_id = $1 AND tombstoned_at IS NULL
			`, account.SubjectID, qualificationOrganizationID, now)
			if err != nil || result.RowsAffected() != 1 {
				return fmt.Errorf("align OIDC qualification profile: rows=%d error=%v", result.RowsAffected(), err)
			}
			result, err = tx.Exec(ctx, `
				UPDATE identity_references SET issuer = $2
				WHERE subject_id = $1 AND email = $3 AND tombstoned_at IS NULL AND deactivated_at IS NULL
			`, account.SubjectID, issuer, account.Email)
			if err != nil || result.RowsAffected() != 1 {
				return fmt.Errorf("bind OIDC qualification issuer: rows=%d error=%v", result.RowsAffected(), err)
			}
		}
		return nil
	})
}

func qualificationAuthorityFor(
	account ProviderAccount,
	revision int64,
	subjectID,
	organizationID string,
	roles []string,
	state string,
) (string, []string, error) {
	if revision != 2 || subjectID != account.SubjectID ||
		strings.TrimSpace(organizationID) == "" || len(roles) != 1 ||
		!containsRole(roles[0]) || state == "ACTIVE" {
		return "", nil, fmt.Errorf(
			"OIDC qualification membership differs from exact predecessor",
		)
	}
	return account.OrganizationID, []string{account.Role}, nil
}

// QualifyExistingProviderAccounts makes the exact existing synthetic users
// login-capable without creating users or changing roles/organizations.
func (endpoint *KeycloakEndpoint) QualifyExistingProviderAccounts(
	ctx context.Context,
	accounts []ProviderAccount,
	password string,
) error {
	if len(accounts) != 9 || len(password) < 24 || strings.TrimSpace(password) != password {
		return fmt.Errorf("bounded OIDC qualification accounts and password are required")
	}
	if err := endpoint.ReconcileProviderAccounts(ctx, accounts); err != nil {
		return fmt.Errorf("preflight existing OIDC qualification accounts: %w", err)
	}
	token, err := endpoint.accessToken(ctx)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		user := keycloakUserForAccount(account)
		user.EmailVerified = true
		user.RequiredActions = []string{}
		response, err := endpoint.doJSON(ctx, http.MethodPut, endpoint.adminURL("users", account.SubjectID), token, user, http.StatusNoContent)
		if err != nil {
			return fmt.Errorf("qualify existing synthetic Keycloak user: %w", err)
		}
		response.Body.Close()
		credential := struct {
			Type      string `json:"type"`
			Value     string `json:"value"`
			Temporary bool   `json:"temporary"`
		}{Type: "password", Value: password, Temporary: false}
		response, err = endpoint.doJSON(ctx, http.MethodPut, endpoint.adminURL("users", account.SubjectID, "reset-password"), token, credential, http.StatusNoContent)
		if err != nil {
			return fmt.Errorf("set existing synthetic Keycloak qualification credential: %w", err)
		}
		response.Body.Close()
		actual, found, err := endpoint.readUser(ctx, token, account.SubjectID)
		if err != nil || !found || !sameQualifiedKeycloakUser(actual, user) {
			return fmt.Errorf("reconcile qualified synthetic Keycloak user")
		}
		mapped, err := endpoint.userRoles(ctx, token, account.SubjectID)
		if err != nil {
			return err
		}
		approved := approvedScenarioRoles(mapped)
		if len(approved) != 1 || approved[0] != account.Role {
			return fmt.Errorf("qualified synthetic Keycloak role changed")
		}
	}
	return nil
}

func sameQualifiedKeycloakUser(left, right scenarioKeycloakUserRepresentation) bool {
	return left.ID == right.ID && left.Username == right.Username && left.Email == right.Email &&
		left.FirstName == right.FirstName && left.LastName == right.LastName && left.Enabled &&
		left.EmailVerified && len(left.RequiredActions) == 0 &&
		len(left.Attributes["organization_id"]) == 1 && len(right.Attributes["organization_id"]) == 1 &&
		left.Attributes["organization_id"][0] == right.Attributes["organization_id"][0]
}
