package qualificationbootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const manifestSchemaVersion = 1

var purposeTokenPattern = regexp.MustCompile(`^[A-Z][A-Z0-9-]{1,63}$`)

type RosterManifest struct {
	SchemaVersion     int             `json:"schemaVersion"`
	ManifestVersion   string          `json:"manifestVersion"`
	AdvisoryLockKey   int64           `json:"advisoryLockKey"`
	Target            string          `json:"target"`
	Enabled           bool            `json:"enabled"`
	QualificationOnly bool            `json:"qualificationOnly"`
	OnboardingMode    string          `json:"onboardingMode"`
	CredentialCustody string          `json:"credentialCustody"`
	Accounts          []RosterAccount `json:"accounts"`
}

type RosterAccount struct {
	PurposeToken          string              `json:"purposeToken"`
	DisplayName           string              `json:"displayName"`
	Email                 string              `json:"email"`
	OrganizationID        string              `json:"organizationId"`
	Role                  string              `json:"role"`
	MembershipID          string              `json:"membershipId"`
	Department            *DepartmentBinding  `json:"departmentMembership,omitempty"`
	AdditionalDepartments []DepartmentBinding `json:"additionalDepartmentMemberships,omitempty"`
}

type DepartmentBinding struct {
	ID                   string `json:"id"`
	DepartmentID         string `json:"departmentId"`
	OrganizationalUnitID string `json:"organizationalUnitId"`
}

type FoundationManifest struct {
	SchemaVersion                    int                      `json:"schemaVersion"`
	ManifestVersion                  string                   `json:"manifestVersion"`
	AdvisoryLockKey                  int64                    `json:"advisoryLockKey"`
	Target                           string                   `json:"target"`
	Enabled                          bool                     `json:"enabled"`
	QualificationOnly                bool                     `json:"qualificationOnly"`
	TargetOrganization               FoundationOrganization   `json:"targetOrganization"`
	AdditionalTargetOrganizations    []FoundationOrganization `json:"additionalTargetOrganizations,omitempty"`
	ControlOrganization              FoundationOrganization   `json:"controlOrganization"`
	ProviderScope                    FoundationScope          `json:"providerScope"`
	AdditionalProviderScopes         []FoundationScope        `json:"additionalProviderScopes,omitempty"`
	RegulatedTarget                  FoundationTarget         `json:"regulatedTarget"`
	AdditionalRegulatedTargets       []FoundationTarget       `json:"additionalRegulatedTargets,omitempty"`
	ControlMustHaveNoProviderScope   bool                     `json:"controlMustHaveNoProviderScope"`
	ControlMustHaveNoRegulatedTarget bool                     `json:"controlMustHaveNoRegulatedTarget"`
}

type FoundationOrganization struct {
	ID               string `json:"id"`
	LegalName        string `json:"legalName"`
	OrganizationType string `json:"organizationType"`
	Status           string `json:"status"`
}

type FoundationScope struct {
	ID                      string   `json:"id"`
	OrganizationID          string   `json:"organizationId"`
	ServiceProviderTypeID   string   `json:"serviceProviderTypeId"`
	AuthorizationIdentifier string   `json:"authorizationIdentifier"`
	Status                  string   `json:"status"`
	PrimaryTargetID         string   `json:"primaryTargetId"`
	TargetIDs               []string `json:"targetIds,omitempty"`
}

type FoundationTarget struct {
	ID                  string  `json:"id"`
	TargetKind          string  `json:"targetKind"`
	OrganizationID      string  `json:"organizationId"`
	OwnerOrganizationID string  `json:"ownerOrganizationId,omitempty"`
	ExternalIdentifier  *string `json:"externalIdentifier"`
}

func ReadRosterManifest(path, expectedDigest, target string) (RosterManifest, string, error) {
	var manifest RosterManifest
	digest, err := readStrictManifest(path, expectedDigest, target, &manifest)
	if err != nil {
		return RosterManifest{}, "", err
	}
	if err := validateRosterManifest(manifest); err != nil {
		return RosterManifest{}, "", err
	}
	return manifest, digest, nil
}

func ReadFoundationManifest(path, expectedDigest, target string) (FoundationManifest, string, error) {
	var manifest FoundationManifest
	digest, err := readStrictManifest(path, expectedDigest, target, &manifest)
	if err != nil {
		return FoundationManifest{}, "", err
	}
	if err := validateFoundationManifest(manifest); err != nil {
		return FoundationManifest{}, "", err
	}
	return manifest, digest, nil
}

func readStrictManifest(path, expectedDigest, target string, destination any) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("bootstrap manifest path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 2 || info.Size() > 2*1024*1024 {
		return "", fmt.Errorf("bootstrap manifest must be a bounded regular non-symlink file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read bootstrap manifest: %w", err)
	}
	digestBytes := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	if strings.TrimSpace(expectedDigest) != digest {
		return "", fmt.Errorf("bootstrap manifest digest mismatch")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return "", fmt.Errorf("decode bootstrap manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return "", fmt.Errorf("bootstrap manifest contains trailing JSON")
	}
	if common, ok := destination.(interface{ getSchemaVersion() int }); ok && common.getSchemaVersion() != manifestSchemaVersion {
		return "", fmt.Errorf("bootstrap manifest schema version is unsupported")
	}
	encoded := strings.ToLower(string(data))
	for _, marker := range []string{"password", "secretbytes", "tokenbytes", "authorizationheader"} {
		if strings.Contains(encoded, marker) {
			return "", fmt.Errorf("bootstrap manifest contains secret-shaped data")
		}
	}
	if !strings.Contains(string(data), `"target":"`+target+`"`) && !strings.Contains(string(data), `"target": "`+target+`"`) {
		return "", fmt.Errorf("bootstrap manifest target mismatch")
	}
	return digest, nil
}

func (manifest RosterManifest) getSchemaVersion() int     { return manifest.SchemaVersion }
func (manifest FoundationManifest) getSchemaVersion() int { return manifest.SchemaVersion }

func validateRosterManifest(manifest RosterManifest) error {
	if manifest.SchemaVersion != manifestSchemaVersion || strings.TrimSpace(manifest.ManifestVersion) == "" || manifest.AdvisoryLockKey <= 0 || strings.TrimSpace(manifest.Target) == "" {
		return fmt.Errorf("roster manifest identity is invalid")
	}
	if manifest.OnboardingMode != "createAndDirectActivate" && manifest.OnboardingMode != "provisionInvite" && manifest.OnboardingMode != "verifyExisting" {
		return fmt.Errorf("roster manifest onboarding mode is invalid")
	}
	if manifest.OnboardingMode == "createAndDirectActivate" && manifest.CredentialCustody == "none" {
		return fmt.Errorf("direct activation requires explicit credential custody")
	}
	if manifest.OnboardingMode != "createAndDirectActivate" && manifest.CredentialCustody != "none" {
		return fmt.Errorf("non-direct onboarding cannot carry password custody")
	}
	if !manifest.Enabled {
		if len(manifest.Accounts) != 0 {
			return fmt.Errorf("disabled roster manifest cannot contain accounts")
		}
		return nil
	}
	if len(manifest.Accounts) == 0 || len(manifest.Accounts) > 128 {
		return fmt.Errorf("enabled roster manifest has an invalid account count")
	}
	seenPurpose := map[string]struct{}{}
	seenEmail := map[string]struct{}{}
	seenMembership := map[string]struct{}{}
	managerRoleCount := 0
	departmentAuthorityCount := 0
	departmentAuthorityPurpose := ""
	for _, account := range manifest.Accounts {
		address, err := mail.ParseAddress(account.Email)
		if err != nil || address.Address != account.Email || strings.TrimSpace(account.DisplayName) == "" || strings.TrimSpace(account.OrganizationID) == "" || strings.TrimSpace(account.MembershipID) == "" || !purposeTokenPattern.MatchString(account.PurposeToken) {
			return fmt.Errorf("roster account identity is invalid")
		}
		if _, ok := seenPurpose[account.PurposeToken]; ok {
			return fmt.Errorf("roster purpose token is duplicated")
		}
		if _, ok := seenEmail[strings.ToLower(account.Email)]; ok {
			return fmt.Errorf("roster email is duplicated")
		}
		if _, ok := seenMembership[account.MembershipID]; ok {
			return fmt.Errorf("roster membership ID is duplicated")
		}
		departments := departmentBindings(account)
		if account.Role == "manager" {
			managerRoleCount++
		}
		if len(departments) > 0 {
			if account.Role != "manager" {
				return fmt.Errorf("only a manager may have department authority")
			}
			departmentAuthorityCount++
			departmentAuthorityPurpose = account.PurposeToken
			seenDepartments := map[string]struct{}{}
			for _, department := range departments {
				if strings.TrimSpace(department.ID) == "" || strings.TrimSpace(department.DepartmentID) == "" || strings.TrimSpace(department.OrganizationalUnitID) == "" {
					return fmt.Errorf("roster department authority is invalid")
				}
				if _, duplicate := seenDepartments[department.ID]; duplicate {
					return fmt.Errorf("roster department authority is duplicated")
				}
				seenDepartments[department.ID] = struct{}{}
			}
		}
		seenPurpose[account.PurposeToken] = struct{}{}
		seenEmail[strings.ToLower(account.Email)] = struct{}{}
		seenMembership[account.MembershipID] = struct{}{}
	}
	if (managerRoleCount != 1 || departmentAuthorityCount != 1 || managerRoleCount != departmentAuthorityCount || departmentAuthorityPurpose == "") && manifest.Enabled {
		return fmt.Errorf("roster must declare exactly one manager and one department authority")
	}
	return nil
}

func validateFoundationManifest(manifest FoundationManifest) error {
	if manifest.SchemaVersion != manifestSchemaVersion || strings.TrimSpace(manifest.ManifestVersion) == "" || manifest.AdvisoryLockKey <= 0 || strings.TrimSpace(manifest.Target) == "" || !manifest.Enabled {
		return fmt.Errorf("foundation manifest is disabled or invalid")
	}
	targetOrganizations := append([]FoundationOrganization{manifest.TargetOrganization}, manifest.AdditionalTargetOrganizations...)
	orgs := append(append([]FoundationOrganization{}, targetOrganizations...), manifest.ControlOrganization)
	seen := map[string]struct{}{}
	for _, org := range orgs {
		if strings.TrimSpace(org.ID) == "" || strings.TrimSpace(org.LegalName) == "" || org.Status != "ACTIVE" || org.OrganizationType == "" {
			return fmt.Errorf("foundation organization is invalid")
		}
		if _, ok := seen[org.ID]; ok {
			return fmt.Errorf("foundation organizations must be distinct")
		}
		seen[org.ID] = struct{}{}
	}
	targets := append([]FoundationTarget{manifest.RegulatedTarget}, manifest.AdditionalRegulatedTargets...)
	targetByID := make(map[string]FoundationTarget, len(targets))
	for _, target := range targets {
		if strings.TrimSpace(target.ID) == "" || target.TargetKind == "" {
			return fmt.Errorf("foundation regulated target is invalid")
		}
		if _, duplicate := targetByID[target.ID]; duplicate {
			return fmt.Errorf("foundation regulated target IDs must be distinct")
		}
		targetByID[target.ID] = target
		switch target.TargetKind {
		case "ORGANIZATION":
			if target.OrganizationID == "" || target.OwnerOrganizationID != "" || target.ExternalIdentifier != nil {
				return fmt.Errorf("foundation organization target is invalid")
			}
		case "FACILITY", "LOCATION":
			if target.OrganizationID != "" || target.OwnerOrganizationID == "" || target.ExternalIdentifier == nil || strings.TrimSpace(*target.ExternalIdentifier) == "" {
				return fmt.Errorf("foundation owned target is invalid")
			}
		default:
			return fmt.Errorf("foundation target kind %q is not supported by the qualification manifest", target.TargetKind)
		}
		ownerOrganizationID := target.OrganizationID
		if ownerOrganizationID == "" {
			ownerOrganizationID = target.OwnerOrganizationID
		}
		if ownerOrganizationID == manifest.ControlOrganization.ID || ownerOrganizationID == "" {
			return fmt.Errorf("foundation target owner is invalid")
		}
		if _, ok := seen[ownerOrganizationID]; !ok {
			return fmt.Errorf("foundation target owner is undeclared")
		}
	}
	scopes := append([]FoundationScope{manifest.ProviderScope}, manifest.AdditionalProviderScopes...)
	seenScopes := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if scope.ID == "" || scope.OrganizationID == "" || scope.ServiceProviderTypeID == "" || scope.Status != "ACTIVE" || scope.AuthorizationIdentifier == "" || scope.PrimaryTargetID == "" {
			return fmt.Errorf("foundation provider scope is invalid")
		}
		if scope.OrganizationID == manifest.ControlOrganization.ID {
			return fmt.Errorf("foundation provider scope cannot belong to the control organization")
		}
		if _, ok := seen[scope.OrganizationID]; !ok {
			return fmt.Errorf("foundation provider scope owner is undeclared")
		}
		if _, duplicate := seenScopes[scope.ID]; duplicate {
			return fmt.Errorf("foundation provider scope IDs must be distinct")
		}
		seenScopes[scope.ID] = struct{}{}
		targetIDs := scope.TargetIDs
		if len(targetIDs) == 0 {
			targetIDs = []string{scope.PrimaryTargetID}
		}
		seenTargetIDs := make(map[string]struct{}, len(targetIDs))
		for _, targetID := range targetIDs {
			if targetID == "" {
				return fmt.Errorf("foundation provider scope target is empty")
			}
			if _, duplicate := seenTargetIDs[targetID]; duplicate {
				return fmt.Errorf("foundation provider scope targets must be distinct")
			}
			seenTargetIDs[targetID] = struct{}{}
			target, ok := targetByID[targetID]
			if !ok || targetOwnerOrganizationID(target) != scope.OrganizationID {
				return fmt.Errorf("foundation provider scope target is incompatible")
			}
		}
		if _, ok := seenTargetIDs[scope.PrimaryTargetID]; !ok {
			return fmt.Errorf("foundation provider scope primary target is not linked")
		}
	}
	if !manifest.ControlMustHaveNoProviderScope || !manifest.ControlMustHaveNoRegulatedTarget {
		return fmt.Errorf("foundation control isolation contract is missing")
	}
	return nil
}

func targetOwnerOrganizationID(target FoundationTarget) string {
	if target.OrganizationID != "" {
		return target.OrganizationID
	}
	return target.OwnerOrganizationID
}

func departmentBindings(account RosterAccount) []DepartmentBinding {
	departments := make([]DepartmentBinding, 0, 1+len(account.AdditionalDepartments))
	if account.Department != nil {
		departments = append(departments, *account.Department)
	}
	departments = append(departments, account.AdditionalDepartments...)
	return departments
}
