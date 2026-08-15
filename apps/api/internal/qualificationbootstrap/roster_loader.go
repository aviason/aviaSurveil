package qualificationbootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type observedRosterAccount struct {
	Manifest RosterAccount
	Provider identity.ProviderDirectoryUser
}

// LoadRoster reconciles the provider first and commits the complete Surveil
// identity graph only after every declared account has passed the same
// manifest-bound preflight. It never creates business-domain records.
func LoadRoster(ctx context.Context, pool *database.Pool, provider identity.ProviderAdmin, manifest RosterManifest, manifestDigest, target, issuer, secretDirectory, actorSubjectID string, now time.Time) error {
	if pool == nil || provider == nil || strings.TrimSpace(manifestDigest) == "" || strings.TrimSpace(target) == "" || strings.TrimSpace(actorSubjectID) == "" || now.IsZero() {
		return fmt.Errorf("roster loader requires database, provider, target, manifest digest, actor, and timestamp")
	}
	if !manifest.Enabled || len(manifest.Accounts) == 0 {
		return fmt.Errorf("roster loader received a disabled or empty manifest")
	}
	if manifest.Target != target {
		return fmt.Errorf("roster manifest target mismatch")
	}
	allUsers, err := listCompleteDirectory(ctx, provider)
	if err != nil {
		return err
	}
	byEmail := make(map[string]identity.ProviderDirectoryUser, len(allUsers))
	for _, user := range allUsers {
		key := strings.ToLower(strings.TrimSpace(user.Email))
		if key == "" {
			continue
		}
		if _, exists := byEmail[key]; exists {
			return fmt.Errorf("provider directory contains duplicate normalized email")
		}
		byEmail[key] = user
	}
	observed := make([]observedRosterAccount, 0, len(manifest.Accounts))
	for _, account := range manifest.Accounts {
		user, found := byEmail[strings.ToLower(account.Email)]
		providerUser := identity.ProviderUser{Email: account.Email, DisplayName: account.DisplayName, MembershipID: account.MembershipID, OrganizationID: account.OrganizationID, Roles: []identity.Role{identity.Role(account.Role)}}
		if !found {
			revisioned, ok := provider.(identity.RevisionedProviderAdmin)
			if !ok {
				return fmt.Errorf("provider does not expose revisioned provisioning")
			}
			subjectID, provisionErr := revisioned.ProvisionUserAtRevision(ctx, providerUser, 0, 1)
			if provisionErr != nil {
				return fmt.Errorf("provision %s: %w", account.PurposeToken, provisionErr)
			}
			user, found, err = locateUniqueDirectoryUser(ctx, provider, account.Email)
			if err != nil {
				return err
			}
			if !found || user.SubjectID != subjectID {
				return fmt.Errorf("provider subject for %s was not stable after provisioning", account.PurposeToken)
			}
		}
		if err := validateProviderIdentity(user, account); err != nil {
			return err
		}
		if manifest.OnboardingMode == "createAndDirectActivate" {
			password, readErr := rosterCredential(secretDirectory, account.PurposeToken)
			if readErr != nil {
				return readErr
			}
			if user.State == "INVITED" {
				authority, ok := provider.(identity.AuthorityRevisionedProviderAdmin)
				if !ok {
					return fmt.Errorf("provider does not expose authority-bound direct activation")
				}
				if err := authority.ActivateUserAtAuthorityRevision(ctx, user.SubjectID, user.MembershipID, user.MembershipRevision, user.MembershipRevision, user.AuthRevision, user.AuthRevision+1, password); err != nil {
					return fmt.Errorf("activate %s: %w", account.PurposeToken, err)
				}
				user, found, err = locateUniqueDirectoryUser(ctx, provider, account.Email)
				if err != nil {
					return err
				}
				if !found {
					return fmt.Errorf("provider account %s disappeared after activation", account.PurposeToken)
				}
			}
			verifier, ok := provider.(identity.CredentialVerifier)
			if !ok {
				return fmt.Errorf("provider does not expose side-effect-free credential verification")
			}
			valid, verifyErr := verifier.VerifyUserCredential(ctx, user.SubjectID, password)
			if verifyErr != nil || !valid {
				return fmt.Errorf("credential verification failed for %s", account.PurposeToken)
			}
		}
		if err := validateProviderAccount(user, account, manifest.OnboardingMode); err != nil {
			return err
		}
		observed = append(observed, observedRosterAccount{Manifest: account, Provider: user})
	}
	return persistRoster(ctx, pool, observed, manifest, manifestDigest, issuer, actorSubjectID, now)
}

func listCompleteDirectory(ctx context.Context, provider identity.ProviderAdmin) ([]identity.ProviderDirectoryUser, error) {
	var users []identity.ProviderDirectoryUser
	first := 0
	for pageNumber := 0; pageNumber < 128; pageNumber++ {
		page, err := provider.ListDirectory(ctx, identity.ProviderDirectoryQuery{First: first, Limit: 100})
		if err != nil {
			return nil, fmt.Errorf("list complete provider directory: %w", err)
		}
		users = append(users, page.Users...)
		if page.NextFirst == 0 {
			return users, nil
		}
		if page.NextFirst <= first {
			return nil, fmt.Errorf("provider directory pagination did not advance")
		}
		first = page.NextFirst
	}
	return nil, fmt.Errorf("provider directory pagination exceeded bounded page count")
}

func locateUniqueDirectoryUser(ctx context.Context, provider identity.ProviderAdmin, email string) (identity.ProviderDirectoryUser, bool, error) {
	page, err := provider.ListDirectory(ctx, identity.ProviderDirectoryQuery{First: 0, Limit: 100, Search: email})
	if err != nil {
		return identity.ProviderDirectoryUser{}, false, err
	}
	var found *identity.ProviderDirectoryUser
	for _, user := range page.Users {
		if strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(email)) {
			if found != nil {
				return identity.ProviderDirectoryUser{}, false, fmt.Errorf("provider directory has duplicate email %q", email)
			}
			copy := user
			found = &copy
		}
	}
	if found == nil {
		return identity.ProviderDirectoryUser{}, false, nil
	}
	return *found, true, nil
}

func validateProviderAccount(user identity.ProviderDirectoryUser, account RosterAccount, mode string) error {
	if err := validateProviderIdentity(user, account); err != nil {
		return err
	}
	if mode == "createAndDirectActivate" {
		if user.State != "ACTIVE" || !user.Enabled || user.Locked || user.TOTPConfigured || len(user.RequiredActions) != 0 {
			return fmt.Errorf("direct-activation account %s is not login-ready: state=%s enabled=%t locked=%t totp=%t requiredActions=%d", account.PurposeToken, user.State, user.Enabled, user.Locked, user.TOTPConfigured, len(user.RequiredActions))
		}
	} else if mode == "provisionInvite" {
		if user.State != "INVITED" || user.Enabled {
			return fmt.Errorf("invited account %s is not pending onboarding", account.PurposeToken)
		}
	} else if mode == "verifyExisting" {
		if user.State != "ACTIVE" || !user.Enabled || user.Locked || !user.TOTPConfigured {
			return fmt.Errorf("existing account %s does not satisfy production MFA policy", account.PurposeToken)
		}
	}
	return nil
}

func validateProviderIdentity(user identity.ProviderDirectoryUser, account RosterAccount) error {
	if strings.TrimSpace(user.SubjectID) == "" || !strings.EqualFold(user.Email, account.Email) || user.DisplayName != account.DisplayName || user.OrganizationID != account.OrganizationID || user.MembershipID != account.MembershipID || user.MembershipRevision != 1 || len(user.Roles) != 1 || string(user.Roles[0]) != account.Role {
		return fmt.Errorf("provider authority drifted for %s", account.PurposeToken)
	}
	return nil
}

func rosterCredential(directory, purpose string) (string, error) {
	if strings.TrimSpace(directory) == "" {
		return "", fmt.Errorf("direct roster activation requires a credential directory")
	}
	root, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, purpose)
	if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("credential path escapes roster directory")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 1 || info.Size() > 4096 {
		return "", fmt.Errorf("credential file for %s is invalid", purpose)
	}
	if info.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("credential file for %s is not private", purpose)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(bytes))
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("credential file for %s is empty or malformed", purpose)
	}
	return value, nil
}

func persistRoster(ctx context.Context, pool *database.Pool, observed []observedRosterAccount, manifest RosterManifest, manifestDigest, issuer, actor string, now time.Time) error {
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, manifest.AdvisoryLockKey); err != nil {
			return fmt.Errorf("lock roster reconciliation: %w", err)
		}
		if err := ensureBootstrapActor(ctx, tx, actor, now); err != nil {
			return err
		}
		for _, account := range observed {
			if err := persistRosterAccount(ctx, tx, account, manifest.OnboardingMode, issuer, actor, now); err != nil {
				return err
			}
		}
		var managerCount int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM caa_department_memberships membership JOIN identity_references identity ON identity.subject_id=membership.subject_id WHERE membership.status='ACTIVE' AND membership.department_id='AERODROME_INSPECTORATE' AND identity.subject_id = ANY($1::text[])`, subjectIDs(observed)).Scan(&managerCount); err != nil {
			return err
		}
		expectedManagerCount := 0
		for _, item := range observed {
			if item.Manifest.Role == "manager" && (manifest.OnboardingMode == "createAndDirectActivate" || manifest.OnboardingMode == "verifyExisting") {
				expectedManagerCount++
			}
		}
		if managerCount != expectedManagerCount {
			return fmt.Errorf("department manager authority count drifted")
		}
		return nil
	})
}

func persistRosterAccount(ctx context.Context, tx pgx.Tx, account observedRosterAccount, mode, issuer, actor string, now time.Time) error {
	user := account.Provider
	requestID := "roster:" + account.Manifest.MembershipID
	state := "ACTIVE"
	providerEnabled := user.Enabled
	if mode == "provisionInvite" {
		state = "INVITED"
		providerEnabled = false
	}
	var identityCount, profileCount, settingsCount, lifecycleCount, versionCount, syncCount, departmentCount int
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM identity_references WHERE subject_id=$1 AND lower(email)=lower($2) AND tombstoned_at IS NULL),(SELECT count(*) FROM user_profiles WHERE subject_id=$1 AND display_name=$3 AND organization_id=$4 AND tombstoned_at IS NULL),(SELECT count(*) FROM user_settings WHERE subject_id=$1),(SELECT count(*) FROM user_lifecycle_requests WHERE id=$5),(SELECT count(*) FROM desired_membership_versions WHERE membership_id=$6 AND subject_id=$1 AND revision=1),(SELECT count(*) FROM desired_membership_sync WHERE membership_id=$6 AND subject_id=$1 AND desired_revision=1),(SELECT count(*) FROM caa_department_memberships WHERE id=$7 AND subject_id=$1 AND status='ACTIVE')`, user.SubjectID, account.Manifest.Email, account.Manifest.DisplayName, account.Manifest.OrganizationID, requestID, account.Manifest.MembershipID, departmentID(account.Manifest)).Scan(&identityCount, &profileCount, &settingsCount, &lifecycleCount, &versionCount, &syncCount, &departmentCount); err != nil {
		return fmt.Errorf("inspect roster state %s: %w", account.Manifest.PurposeToken, err)
	}
	expectedDepartment := account.Manifest.Department != nil && mode != "provisionInvite"
	allPresent := identityCount == 1 && profileCount == 1 && settingsCount == 1 && lifecycleCount == 1 && versionCount == 1 && syncCount == 1 && ((departmentCount == 1) == expectedDepartment)
	anyPresent := identityCount+profileCount+settingsCount+lifecycleCount+versionCount+syncCount+departmentCount > 0
	if allPresent {
		return verifyPersistedRosterAccount(ctx, tx, account, mode, requestID, actor)
	}
	if anyPresent {
		return fmt.Errorf("roster application state is partial for %s", account.Manifest.PurposeToken)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO identity_references (subject_id,issuer,display_name,revision,email,created_at) VALUES ($1,$2,$3,1,$4,$5)`, user.SubjectID, issuer, account.Manifest.DisplayName, account.Manifest.Email, now); err != nil {
		return fmt.Errorf("insert roster identity %s: %w", account.Manifest.PurposeToken, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_profiles (subject_id,display_name,organization_id,revision,created_at,updated_at) VALUES ($1,$2,$3,1,$4,$4)`, user.SubjectID, account.Manifest.DisplayName, account.Manifest.OrganizationID, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_settings (subject_id,notification_preferences,locale,timezone,revision,updated_at) VALUES ($1,'{}'::jsonb,'en','UTC',1,$2)`, user.SubjectID, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_lifecycle_requests (id,subject_id,requested_action,requested_roles,requested_organization_id,status,idempotency_key,requested_by_subject_id,requested_email,requested_display_name,expected_membership_revision,reason,requested_effective_at,membership_id,resulting_membership_revision,provider_acknowledged_at,created_at,updated_at) VALUES ($1,$2,'PROVISION',$3,$4,'SUCCEEDED',$5,$6,$7,$8,0,$9,$10,$11,1,$10,$10,$10)`, requestID, user.SubjectID, []string{account.Manifest.Role}, account.Manifest.OrganizationID, requestID, actor, account.Manifest.Email, account.Manifest.DisplayName, "Target-bound prepared roster reconciliation", now, account.Manifest.MembershipID); err != nil {
		return fmt.Errorf("insert roster lifecycle %s: %w", account.Manifest.PurposeToken, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO desired_membership_versions (membership_id,subject_id,revision,membership_state,organization_id,roles,requested_by_subject_id,reason,source_request_id,requested_at,effective_at,observed_provider_enabled,observed_organization_id,observed_roles,observed_at,drift_state) VALUES ($1,$2,1,$3,$4,$5,$6,$7,$8,$9,$9,$10,$4,$5,$9,'IN_SYNC')`, account.Manifest.MembershipID, user.SubjectID, state, account.Manifest.OrganizationID, []string{account.Manifest.Role}, actor, "Target-bound prepared roster reconciliation", requestID, now, providerEnabled); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO desired_membership_sync (membership_id,subject_id,desired_revision,observed_provider_enabled,observed_organization_id,observed_roles,observed_at,drift_state) VALUES ($1,$2,1,$3,$4,$5,$6,'IN_SYNC')`, account.Manifest.MembershipID, user.SubjectID, providerEnabled, account.Manifest.OrganizationID, []string{account.Manifest.Role}, now); err != nil {
		return err
	}
	if expectedDepartment {
		department := account.Manifest.Department
		if _, err := tx.Exec(ctx, `INSERT INTO caa_department_memberships (id,subject_id,department_id,organizational_unit_id,membership_role,effective_from,status,created_at,root_id) VALUES ($1,$2,$3,$4,'DEPARTMENT_MANAGER',$5::date,'ACTIVE',$6,$1)`, department.ID, user.SubjectID, department.DepartmentID, department.OrganizationalUnitID, now.Format(time.DateOnly), now); err != nil {
			return err
		}
	}
	return nil
}

func verifyPersistedRosterAccount(ctx context.Context, tx pgx.Tx, account observedRosterAccount, mode, requestID, actor string) error {
	var state, organization, observedOrganization, drift string
	var roles []string
	var enabled bool
	if err := tx.QueryRow(ctx, `SELECT membership_state,organization_id,roles,observed_provider_enabled,observed_organization_id,drift_state FROM desired_membership_versions WHERE membership_id=$1 AND subject_id=$2 AND revision=1`, account.Manifest.MembershipID, account.Provider.SubjectID).Scan(&state, &organization, &roles, &enabled, &observedOrganization, &drift); err != nil {
		return err
	}
	expectedState := "ACTIVE"
	if mode == "provisionInvite" {
		expectedState = "INVITED"
	}
	if state != expectedState || organization != account.Manifest.OrganizationID || observedOrganization != account.Manifest.OrganizationID || len(roles) != 1 || roles[0] != account.Manifest.Role || drift != "IN_SYNC" {
		return fmt.Errorf("persisted roster authority drifted for %s", account.Manifest.PurposeToken)
	}
	_ = requestID
	_ = actor
	_ = enabled
	return nil
}

func subjectIDs(accounts []observedRosterAccount) []string {
	result := make([]string, 0, len(accounts))
	for _, account := range accounts {
		result = append(result, account.Provider.SubjectID)
	}
	return result
}
func departmentID(account RosterAccount) string {
	if account.Department == nil {
		return ""
	}
	return account.Department.ID
}
