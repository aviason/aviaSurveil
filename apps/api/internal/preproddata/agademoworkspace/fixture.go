package agademoworkspace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var connectedProviderCodes = []string{
	"AERODROME_OPERATOR", "ANSP", "CNS_PROVIDER", "AIS_AIM_PROVIDER", "MET_PROVIDER", "SAR_ORGANIZATION", "AVSEC_PROVIDER", "AIR_OPERATOR", "AMO", "ATO", "GROUND_HANDLING", "FUEL_PROVIDER", "CARGO_REGULATED_AGENT", "RPAS_UAS_OPERATOR",
}

var noDefaultProviderCodes = []string{"CAMO", "FSTD", "DOA", "POA", "AEMC", "AME"}

type ProviderConfigurationEntry struct {
	ProviderTypeCode string `json:"providerTypeCode"`
	Disposition      string `json:"disposition"`
	ReasonCode       string `json:"reasonCode"`
}

type FixtureRoleSlot struct {
	Slot          string   `json:"slot"`
	RequiredRoles []string `json:"requiredRoles"`
	Projection    string   `json:"projection"`
}

type FixtureOrganization struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Other          bool   `json:"other"`
}

type FixtureScopeTemplate struct {
	ScopeSlot            string `json:"scopeSlot"`
	OrganizationSlot     string `json:"organizationSlot"`
	ProviderTypeCode     string `json:"providerTypeCode"`
	ProviderScopeRootID  string `json:"providerScopeRootId"`
	ProviderScopeID      string `json:"providerScopeId"`
	ProviderScopeVersion int    `json:"providerScopeVersion"`
	TargetID             string `json:"targetId"`
	CanonicalTargetKind  string `json:"canonicalTargetKind"`
	TargetProfileCode    string `json:"targetProfileCode"`
}

type FixtureTemplate struct {
	SchemaVersion         string                       `json:"schemaVersion"`
	TemplateID            string                       `json:"templateId"`
	CAAOrganizationID     string                       `json:"caaOrganizationId"`
	DepartmentID          string                       `json:"departmentId"`
	OrganizationalUnitID  string                       `json:"organizationalUnitId"`
	SyntheticNamespace    string                       `json:"syntheticNamespace"`
	RoleSlots             []FixtureRoleSlot            `json:"roleSlots"`
	Organizations         []FixtureOrganization        `json:"organizations"`
	ProviderConfiguration []ProviderConfigurationEntry `json:"providerConfiguration"`
	Scopes                []FixtureScopeTemplate       `json:"scopes"`
	BindingRules          []AuthorityBinding           `json:"bindingRules"`
}

type FixtureAccount struct {
	Slot              string   `json:"slot"`
	SubjectID         string   `json:"subjectId"`
	MembershipID      string   `json:"membershipId"`
	MembershipVersion int      `json:"membershipVersion"`
	OrganizationID    string   `json:"organizationId"`
	Roles             []string `json:"roles"`
	MembershipDigest  string   `json:"membershipDigest"`
}

type FixtureManifest struct {
	SchemaVersion           string                       `json:"schemaVersion"`
	ManifestID              string                       `json:"manifestId"`
	TemplateID              string                       `json:"templateId"`
	TemplateDigest          string                       `json:"templateDigest"`
	TargetFingerprintDigest string                       `json:"targetFingerprintDigest"`
	BaseRunID               string                       `json:"baseRunId"`
	ProviderCatalogDigest   string                       `json:"providerCatalogDigest"`
	ProviderConfiguration   []ProviderConfigurationEntry `json:"providerConfiguration"`
	Accounts                []FixtureAccount             `json:"accounts"`
	Bindings                []AuthorityBinding           `json:"bindings"`
	ExportedAt              time.Time                    `json:"exportedAt"`
	ManifestDigest          string                       `json:"manifestDigest"`
}

type FixtureSource interface {
	ReadExactAccounts(context.Context, []string) ([]FixtureAccount, error)
}

type FixtureSourceFunc func(context.Context, []string) ([]FixtureAccount, error)

func (function FixtureSourceFunc) ReadExactAccounts(ctx context.Context, slots []string) ([]FixtureAccount, error) {
	return function(ctx, slots)
}

func DefaultFixtureTemplate() FixtureTemplate {
	roles := []FixtureRoleSlot{
		{Slot: "CAA_ADMIN", RequiredRoles: []string{"admin"}, Projection: "CAA_FULL"},
		{Slot: "DEPARTMENT_MANAGER", RequiredRoles: []string{"manager"}, Projection: "CLASSIFICATION_AND_LIFECYCLE_MANAGER"},
		{Slot: "INSPECTOR", RequiredRoles: []string{"inspector"}, Projection: "ASSIGNED_INSPECTOR"},
		{Slot: "LEAD_INSPECTOR", RequiredRoles: []string{"lead_inspector"}, Projection: "ASSIGNED_LEAD"},
		{Slot: "CAA_REVIEWER", RequiredRoles: []string{"lead_inspector"}, Projection: "CAP_REVIEWER"},
		{Slot: "AUDITEE_MATCHING", RequiredRoles: []string{"auditee"}, Projection: "MATCHING_ORGANIZATION"},
		{Slot: "AUDITEE_OTHER_ORGANIZATION", RequiredRoles: []string{"auditee"}, Projection: "OTHER_ORGANIZATION"},
		{Slot: "CAA_UNRELATED", RequiredRoles: []string{"caa_reviewer"}, Projection: "UNRELATED_CAA"},
		{Slot: "INSPECTOR_OTHER", RequiredRoles: []string{"inspector"}, Projection: "OTHER_SCOPE"},
	}
	organizations := []FixtureOrganization{{OrganizationID: "AGA-DEMO-CAA", Name: "Synthetic CAA", Other: false}, {OrganizationID: "AGA-DEMO-OTHER-ORG", Name: "Synthetic Other Organization", Other: true}}
	provider := make([]ProviderConfigurationEntry, 0, len(connectedProviderCodes)+len(noDefaultProviderCodes))
	for _, code := range connectedProviderCodes {
		disposition := "INSPECTED_SCOPE_ELIGIBLE"
		if code != "AERODROME_OPERATOR" {
			disposition = "INVOLVEMENT_ONLY"
		}
		provider = append(provider, ProviderConfigurationEntry{ProviderTypeCode: code, Disposition: disposition, ReasonCode: "CANDIDATE_PROFILE_BOUNDARY"})
	}
	for _, code := range noDefaultProviderCodes {
		provider = append(provider, ProviderConfigurationEntry{ProviderTypeCode: code, Disposition: "NO_DEFAULT_AGA_RELATIONSHIP", ReasonCode: "NO_PROFILE_AUTHORITY"})
	}
	scopes := []FixtureScopeTemplate{
		{ScopeSlot: "MATCHING_AERODROME_OPERATOR", OrganizationSlot: "MATCHING", ProviderTypeCode: "AERODROME_OPERATOR", ProviderScopeRootID: "aga-ws-scope-root-matching", ProviderScopeID: "aga-ws-scope-matching", ProviderScopeVersion: 1, TargetID: "aga-ws-target-matching", CanonicalTargetKind: "FACILITY", TargetProfileCode: "RFFS_FUNCTION"},
		{ScopeSlot: "OTHER_AERODROME_OPERATOR", OrganizationSlot: "OTHER", ProviderTypeCode: "AERODROME_OPERATOR", ProviderScopeRootID: "aga-ws-scope-root-other", ProviderScopeID: "aga-ws-scope-other", ProviderScopeVersion: 1, TargetID: "aga-ws-target-other", CanonicalTargetKind: "FACILITY", TargetProfileCode: "RFFS_FUNCTION"},
	}
	bindings := make([]AuthorityBinding, 0, len(roles))
	for _, role := range roles {
		organization := "AGA-DEMO-CAA"
		if role.Slot == "AUDITEE_OTHER_ORGANIZATION" || role.Slot == "INSPECTOR_OTHER" {
			organization = "AGA-DEMO-OTHER-ORG"
		}
		bindings = append(bindings, AuthorityBinding{BindingID: "template-" + strings.ToLower(role.Slot), SubjectSlot: role.Slot, MembershipSlot: role.Slot + "_MEMBERSHIP", OrganizationID: organization, DepartmentID: "AGA-DEMO-DEPARTMENT", OrganizationalUnitID: "AGA-DEMO-UNIT", OperationRoles: append([]string(nil), role.RequiredRoles...), Active: true})
	}
	return FixtureTemplate{SchemaVersion: FixtureSchemaVersion, TemplateID: "aga-demo-authority-template-v1", CAAOrganizationID: "AGA-DEMO-CAA", DepartmentID: "AGA-DEMO-DEPARTMENT", OrganizationalUnitID: "AGA-DEMO-UNIT", SyntheticNamespace: "AGA_DEMO_ONLY", RoleSlots: roles, Organizations: organizations, ProviderConfiguration: provider, Scopes: scopes, BindingRules: bindings}
}

func (template FixtureTemplate) Validate() error {
	if template.SchemaVersion != FixtureSchemaVersion || template.TemplateID == "" || template.CAAOrganizationID == "" || template.DepartmentID == "" || template.OrganizationalUnitID == "" || template.SyntheticNamespace != "AGA_DEMO_ONLY" {
		return ErrWorkspaceFixture
	}
	if len(template.RoleSlots) != 9 || len(template.Organizations) != 2 || len(template.ProviderConfiguration) != 20 || len(template.Scopes) != 2 || len(template.BindingRules) != 9 {
		return ErrWorkspaceFixture
	}
	seen := make(map[string]bool)
	for _, role := range template.RoleSlots {
		if role.Slot == "" || seen[role.Slot] || len(role.RequiredRoles) == 0 {
			return ErrWorkspaceFixture
		}
		seen[role.Slot] = true
	}
	provider := make(map[string]string)
	for _, entry := range template.ProviderConfiguration {
		if entry.ProviderTypeCode == "" || provider[entry.ProviderTypeCode] != "" {
			return ErrWorkspaceFixture
		}
		provider[entry.ProviderTypeCode] = entry.Disposition
	}
	for _, code := range connectedProviderCodes {
		if provider[code] == "" {
			return ErrWorkspaceFixture
		}
	}
	for _, code := range noDefaultProviderCodes {
		if provider[code] != "NO_DEFAULT_AGA_RELATIONSHIP" {
			return ErrWorkspaceFixture
		}
	}
	return nil
}

func (template FixtureTemplate) Digest() string {
	return digestValue("AGA-DEMO-WORKSPACE-FIXTURE-TEMPLATE-V1", template)
}

func (manifest FixtureManifest) Validate() error {
	if manifest.SchemaVersion != FixtureSchemaVersion || manifest.ManifestID == "" || manifest.TemplateID == "" || !validDigest(manifest.TemplateDigest) || !validDigest(manifest.TargetFingerprintDigest) || !validDigest(manifest.ProviderCatalogDigest) || manifest.ExportedAt.IsZero() || !validDigest(manifest.ManifestDigest) {
		return ErrWorkspaceFixture
	}
	if len(manifest.ProviderConfiguration) != 20 || len(manifest.Accounts) != 9 || len(manifest.Bindings) != 9 {
		return ErrWorkspaceFixture
	}
	providerCodes := make(map[string]string, len(manifest.ProviderConfiguration))
	for _, entry := range manifest.ProviderConfiguration {
		if entry.ProviderTypeCode == "" || providerCodes[entry.ProviderTypeCode] != "" {
			return ErrWorkspaceFixture
		}
		providerCodes[entry.ProviderTypeCode] = entry.Disposition
	}
	seen := make(map[string]bool)
	for _, account := range manifest.Accounts {
		if account.Slot == "" || seen[account.Slot] || account.SubjectID == "" || account.MembershipID == "" || account.MembershipVersion < 1 || account.OrganizationID == "" || !validDigest(account.MembershipDigest) {
			return ErrWorkspaceFixture
		}
		seen[account.Slot] = true
	}
	return nil
}

func ExportFixture(ctx context.Context, template FixtureTemplate, source FixtureSource, manifestID, targetDigest, baseRunID, providerCatalogDigest string, now time.Time) (FixtureManifest, error) {
	if err := template.Validate(); err != nil {
		return FixtureManifest{}, err
	}
	if source == nil || manifestID == "" || !validDigest(targetDigest) || !validDigest(providerCatalogDigest) || baseRunID == "" {
		return FixtureManifest{}, ErrWorkspaceFixture
	}
	slots := make([]string, 0, len(template.RoleSlots))
	for _, slot := range template.RoleSlots {
		slots = append(slots, slot.Slot)
	}
	sort.Strings(slots)
	accounts, err := source.ReadExactAccounts(ctx, slots)
	if err != nil {
		return FixtureManifest{}, err
	}
	if len(accounts) != len(slots) {
		return FixtureManifest{}, ErrWorkspaceFixture
	}
	for _, account := range accounts {
		if account.Slot == "" || account.SubjectID == "" || account.MembershipID == "" || account.MembershipVersion < 1 || !validDigest(account.MembershipDigest) {
			return FixtureManifest{}, ErrWorkspaceFixture
		}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	manifest := FixtureManifest{SchemaVersion: FixtureSchemaVersion, ManifestID: manifestID, TemplateID: template.TemplateID, TemplateDigest: template.Digest(), TargetFingerprintDigest: targetDigest, BaseRunID: baseRunID, ProviderCatalogDigest: providerCatalogDigest, ProviderConfiguration: cloneJSON(template.ProviderConfiguration), Accounts: accounts, Bindings: cloneJSON(template.BindingRules), ExportedAt: now}
	manifest.ManifestDigest = digestValue("AGA-DEMO-WORKSPACE-FIXTURE-MANIFEST-V1", manifest)
	return manifest, nil
}

func VerifyFixture(ctx context.Context, template FixtureTemplate, source FixtureSource, expected FixtureManifest) error {
	if err := template.Validate(); err != nil {
		return err
	}
	if err := expected.Validate(); err != nil {
		return err
	}
	if expected.TemplateDigest != template.Digest() {
		return ErrWorkspaceFixture
	}
	slots := make([]string, 0, len(template.RoleSlots))
	for _, slot := range template.RoleSlots {
		slots = append(slots, slot.Slot)
	}
	sort.Strings(slots)
	accounts, err := source.ReadExactAccounts(ctx, slots)
	if err != nil {
		return err
	}
	if len(accounts) != len(expected.Accounts) {
		return ErrWorkspaceFixture
	}
	got := map[string]FixtureAccount{}
	for _, account := range accounts {
		got[account.Slot] = account
	}
	for _, want := range expected.Accounts {
		actual, ok := got[want.Slot]
		if !ok || actual.SubjectID != want.SubjectID || actual.MembershipID != want.MembershipID || actual.MembershipVersion != want.MembershipVersion || actual.OrganizationID != want.OrganizationID || actual.MembershipDigest != want.MembershipDigest {
			return ErrWorkspaceFixture
		}
	}
	return nil
}

func LoadFixtureTemplate(path string) (FixtureTemplate, error) {
	data, err := readPrivateOrTrackedJSON(path, false)
	if err != nil {
		return FixtureTemplate{}, err
	}
	var template FixtureTemplate
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&template); err != nil {
		return FixtureTemplate{}, fmt.Errorf("decode workspace fixture template: %w", err)
	}
	if err := template.Validate(); err != nil {
		return FixtureTemplate{}, err
	}
	return template, nil
}

func LoadFixtureManifest(path string) (FixtureManifest, error) {
	data, err := readPrivateOrTrackedJSON(path, true)
	if err != nil {
		return FixtureManifest{}, err
	}
	var manifest FixtureManifest
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return FixtureManifest{}, fmt.Errorf("decode workspace fixture manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return FixtureManifest{}, err
	}
	return manifest, nil
}

func WriteFixtureManifest(path string, manifest FixtureManifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("fixture manifest path must be absolute")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return nil
}

func readPrivateOrTrackedJSON(path string, private bool) ([]byte, error) {
	if path == "" {
		return nil, ErrWorkspaceFixture
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrWorkspaceFixture
	}
	if private && info.Mode().Perm() != 0o600 {
		return nil, ErrWorkspaceFixture
	}
	return os.ReadFile(path)
}
