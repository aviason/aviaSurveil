package agademoworkspace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

// NewPostgresRecommendationScopeResolver reads the immutable sealed scope
// projection. The browser supplies exact selectors, while this resolver
// returns the complete provider fact, including the effective interval and
// qualifier maps that are bound into the readiness/profile digest.
func NewPostgresRecommendationScopeResolver(pool *database.Pool) RecommendationScopeResolver {
	return func(ctx context.Context, workspace preprod.LoadedWorkspace, request aga.RecommendationRequest) ([]aga.ProviderScopeFact, error) {
		if pool == nil || workspace.Generation.GenerationID == "" {
			return nil, fmt.Errorf("workspace scope resolver requires a PostgreSQL pool and generation")
		}
		rows, err := pool.Query(ctx, `
			SELECT payload
			FROM preprod_aga_demo_workspace.sealed_provider_scopes
			WHERE generation_id = $1
			  AND organization_id = $2
			  AND provider_scope_root_id = $3
			  AND provider_scope_id = $4
			  AND provider_scope_version = $5
		`, workspace.Generation.GenerationID, request.OrganizationID, request.ProviderScopeRootID, request.ProviderScopeID, request.ProviderScopeVersion)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		facts := make([]aga.ProviderScopeFact, 0, 1)
		for rows.Next() {
			var payload []byte
			if err := rows.Scan(&payload); err != nil {
				return nil, err
			}
			var scope preprod.ProviderScope
			if err := json.Unmarshal(payload, &scope); err != nil {
				return nil, fmt.Errorf("decode sealed provider scope: %w", err)
			}
			fact := aga.ProviderScopeFact{
				GenerationID: scope.GenerationID, ProfileDigest: scope.ProfileDigest,
				OrganizationID: scope.OrganizationID, ProviderScopeRootID: scope.ProviderScopeRootID,
				ProviderScopeID: scope.ProviderScopeID, ProviderScopeVersion: scope.ProviderScopeVersion,
				ProviderTypeID: scope.ProviderTypeID, ProviderTypeCode: scope.ProviderTypeCode,
				Status: scope.Status, EffectiveFrom: scope.EffectiveFrom, EffectiveTo: scope.EffectiveTo,
				DepartmentID: scope.DepartmentID, OrganizationalUnitID: scope.OrganizationalUnitID,
				OperationQualifiers: append([]aga.Qualifier(nil), scope.OperationQualifiers...),
				ActivityQualifiers:  append([]aga.Qualifier(nil), scope.ActivityQualifiers...),
			}
			for _, target := range scope.Targets {
				fact.Targets = append(fact.Targets, aga.TypedTarget{ID: target.TargetID, Kind: target.CanonicalKind, ProfileCode: target.ProfileCode})
			}
			facts = append(facts, fact)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return facts, nil
	}
}

// NewPostgresSimulationSetupResolver returns the one current matching
// provider/target scope and opaque role-selection pins. It reads only sealed
// workspace projections and never creates readiness or recommendation state.
func NewPostgresSimulationSetupResolver(pool *database.Pool) SimulationSetupResolver {
	return func(ctx context.Context, workspace preprod.LoadedWorkspace, principal identity.Principal) (SimulationSetupProjection, error) {
		if pool == nil || workspace.Generation.GenerationID == "" || principal.OrganizationID == "" {
			return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
		}
		rows, err := pool.Query(ctx, `
			SELECT payload
			FROM preprod_aga_demo_workspace.sealed_provider_scopes
			WHERE generation_id = $1 AND provider_type_code = 'AERODROME_OPERATOR'
			ORDER BY provider_scope_id, provider_scope_version
		`, workspace.Generation.GenerationID)
		if err != nil {
			return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
		}
		defer rows.Close()
		var scopes []preprod.ProviderScope
		for rows.Next() {
			var payload []byte
			if err := rows.Scan(&payload); err != nil {
				return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
			}
			var scope preprod.ProviderScope
			if err := json.Unmarshal(payload, &scope); err != nil {
				return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
			}
			scopes = append(scopes, scope)
		}
		scopes = simulationSetupScopesForPrincipal(scopes, principal.OrganizationID)
		if err := rows.Err(); err != nil || len(scopes) != 1 {
			return SimulationSetupProjection{}, ErrRecommendationAmbiguous
		}
		scope := scopes[0]
		if len(scope.Targets) != 1 {
			return SimulationSetupProjection{}, ErrRecommendationAmbiguous
		}
		accounts := make(map[string]preprod.FixtureAccount, len(workspace.Fixture.Accounts))
		for _, account := range workspace.Fixture.Accounts {
			accounts[account.Slot] = account
		}
		choices := func(kind, slot string) ([]SimulationRoleChoice, error) {
			var result []SimulationRoleChoice
			for _, binding := range workspace.Fixture.Bindings {
				if binding.SubjectSlot != slot || !binding.Active || binding.OrganizationID != scope.OrganizationID {
					continue
				}
				account, found := accounts[binding.SubjectSlot]
				if !found || account.SubjectID == "" {
					continue
				}
				fact := LifecycleBindingFact{BindingID: binding.BindingID, BindingRevision: account.MembershipVersion, SubjectID: account.SubjectID, MembershipSlot: binding.MembershipSlot, OrganizationID: binding.OrganizationID, DepartmentID: binding.DepartmentID, OrganizationalUnitID: binding.OrganizationalUnitID, ProviderScopeID: scope.ProviderScopeID, Active: true}
				if fact.BindingRevision < 1 {
					fact.BindingRevision = 1
				}
				result = append(result, SimulationRoleChoice{SelectionPin: selectionPinFor(fact), Label: kind, Role: kind})
			}
			if len(result) != 1 {
				return nil, ErrLifecycleBindingMismatch
			}
			return result, nil
		}
		inspectorChoices, err := choices("Assigned Inspector", "INSPECTOR")
		if err != nil {
			return SimulationSetupProjection{}, err
		}
		leadChoices, err := choices("Assigned Lead Inspector", "LEAD_INSPECTOR")
		if err != nil {
			return SimulationSetupProjection{}, err
		}
		target := scope.Targets[0]
		setup := SimulationSetupProjection{
			GenerationID: workspace.Generation.GenerationID, GenerationRevision: workspace.Generation.Revision, GenerationSealDigest: workspace.Generation.SealDigest,
			DraftID: workspace.Draft.Draft.DraftID, DraftRevision: workspace.Draft.Draft.Revision, DraftContentDigest: workspace.Draft.Draft.ContentDigest,
			TaxonomyVersion: workspace.Generation.TaxonomyVersion, TaxonomyDigest: workspace.Generation.TaxonomyDigest, ClassificationRunID: workspace.Run.Result.ClassificationRunID, ClassificationRunDigest: workspace.Run.ClassificationRunDigest,
			OrganizationLabel: scope.OrganizationID, ProviderLabel: scope.ProviderTypeCode, TargetLabel: target.TargetID,
			ProviderScopeRootID: scope.ProviderScopeRootID, ProviderScopeID: scope.ProviderScopeID, ProviderScopeVersion: scope.ProviderScopeVersion,
			ProviderTypeID: scope.ProviderTypeID, ProviderTypeCode: scope.ProviderTypeCode, DepartmentID: scope.DepartmentID, OrganizationalUnitID: scope.OrganizationalUnitID,
			TargetID: target.TargetID, CanonicalTargetKind: target.CanonicalKind, TargetProfileCode: target.ProfileCode, InspectionProfileCode: "EMERGENCY_AND_RFFS", InspectionTypeCode: "ON_SITE_INSPECTION",
			OperationQualifiers: append([]aga.Qualifier(nil), scope.OperationQualifiers...), ActivityQualifiers: append([]aga.Qualifier(nil), scope.ActivityQualifiers...), EffectiveAt: scope.EffectiveFrom.UTC().Format(time.RFC3339Nano),
			ProviderScopeDigest: scope.ProfileDigest, ReadinessState: string(workspace.Draft.Draft.State), InspectorChoices: inspectorChoices, LeadChoices: leadChoices,
		}
		if len(workspace.Draft.Draft.ReadinessEvents) > 0 {
			readiness := workspace.Draft.Draft.ReadinessEvents[len(workspace.Draft.Draft.ReadinessEvents)-1]
			setup.ReadinessState = string(workspace.Draft.Draft.State)
			setup.ReadinessEventDigest = readiness.ReadinessEventDigest
		}
		setup.SimulationSetupDigest = setupDigest(setup)
		return setup, nil
	}
}

// simulationSetupScopesForPrincipal applies the only synthetic organization
// alias permitted by the connected fixture. The query intentionally loads
// every AERODROME_OPERATOR scope in the sealed generation first, so an
// additional matching scope remains an ambiguity instead of being hidden by
// a caller-supplied organization selector.
func simulationSetupScopesForPrincipal(scopes []preprod.ProviderScope, principalOrganization string) []preprod.ProviderScope {
	filtered := make([]preprod.ProviderScope, 0, len(scopes))
	for _, scope := range scopes {
		if workspaceOrganizationMatchesPrincipal(principalOrganization, scope.OrganizationID) {
			filtered = append(filtered, scope)
		}
	}
	return filtered
}

// NewFixtureLifecycleBindingResolver pins lifecycle actors to the exported
// synthetic fixture. It never resolves canonical assignment or membership
// authority; membership versions are used as the binding revision pin.
func NewFixtureLifecycleBindingResolver() LifecycleBindingResolver {
	return func(_ context.Context, workspace preprod.LoadedWorkspace, recommendation aga.Recommendation, kind string) ([]LifecycleBindingFact, error) {
		type selection struct {
			slot         string
			organization string
		}
		wanted := map[string]selection{
			"INSPECTOR": {slot: "INSPECTOR", organization: recommendation.OrganizationID},
			"LEAD":      {slot: "LEAD_INSPECTOR", organization: recommendation.OrganizationID},
			"AUDITEE":   {slot: "AUDITEE_MATCHING", organization: recommendation.OrganizationID},
		}
		choice, ok := wanted[kind]
		if !ok {
			return nil, fmt.Errorf("unknown lifecycle binding kind %q", kind)
		}
		accounts := make(map[string]preprod.FixtureAccount, len(workspace.Fixture.Accounts))
		for _, account := range workspace.Fixture.Accounts {
			accounts[account.Slot] = account
		}
		facts := make([]LifecycleBindingFact, 0, 1)
		for _, binding := range workspace.Fixture.Bindings {
			if binding.SubjectSlot != choice.slot || binding.OrganizationID != choice.organization || !binding.Active {
				continue
			}
			account, found := accounts[binding.SubjectSlot]
			if !found || account.SubjectID == "" || account.OrganizationID == "" {
				continue
			}
			revision := account.MembershipVersion
			if revision < 1 {
				revision = 1
			}
			facts = append(facts, LifecycleBindingFact{
				BindingID: binding.BindingID, BindingRevision: revision, SubjectID: account.SubjectID,
				MembershipSlot: binding.MembershipSlot, OrganizationID: binding.OrganizationID,
				SourceOrganizationID: account.OrganizationID,
				DepartmentID:         binding.DepartmentID, OrganizationalUnitID: binding.OrganizationalUnitID,
				ProviderScopeID: recommendation.ProviderScopeID, Active: true,
			})
		}
		return facts, nil
	}
}
