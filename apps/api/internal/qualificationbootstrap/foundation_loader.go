package qualificationbootstrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

// LoadFoundation is the sole writer for qualification organizations, scopes,
// and regulated targets. Roster and catalog loaders only verify these rows.
func LoadFoundation(ctx context.Context, pool *database.Pool, manifest FoundationManifest, manifestDigest, actorSubjectID string, now time.Time) error {
	if pool == nil || strings.TrimSpace(manifestDigest) == "" || strings.TrimSpace(actorSubjectID) == "" || now.IsZero() {
		return fmt.Errorf("foundation loader requires database, manifest digest, actor, and timestamp")
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, manifest.AdvisoryLockKey); err != nil {
			return fmt.Errorf("lock foundation reconciliation: %w", err)
		}
		if err := ensureBootstrapActor(ctx, tx, actorSubjectID, now); err != nil {
			return err
		}
		for _, organization := range foundationOrganizations(manifest) {
			if _, err := tx.Exec(ctx, `INSERT INTO organizations (id,legal_name,organization_type,status,revision,created_at,updated_at) VALUES ($1,$2,$3,$4,1,$5,$5) ON CONFLICT (id) DO NOTHING`, organization.ID, organization.LegalName, organization.OrganizationType, organization.Status, now); err != nil {
				return fmt.Errorf("insert foundation organization %s: %w", organization.ID, err)
			}
			var actual FoundationOrganization
			if err := tx.QueryRow(ctx, `SELECT id,legal_name,organization_type,status FROM organizations WHERE id=$1 AND tombstoned_at IS NULL`, organization.ID).Scan(&actual.ID, &actual.LegalName, &actual.OrganizationType, &actual.Status); err != nil {
				return fmt.Errorf("verify foundation organization %s: %w", organization.ID, err)
			}
			if actual != organization {
				return fmt.Errorf("foundation organization %s drifted", organization.ID)
			}
		}
		for _, target := range foundationTargets(manifest) {
			ownerOrganizationID := targetOwnerOrganizationID(target)
			if _, err := tx.Exec(ctx, `INSERT INTO regulated_targets (id,target_kind,organization_id,person_subject_id,owner_organization_id,external_identifier,created_at) VALUES ($1,$2,$3,NULL,$4,$5,$6) ON CONFLICT (id) DO NOTHING`, target.ID, target.TargetKind, nullableString(target.OrganizationID), nullableString(target.OwnerOrganizationID), nullableOptionalString(target.ExternalIdentifier), now); err != nil {
				return fmt.Errorf("insert foundation regulated target %s: %w", target.ID, err)
			}
			var actual FoundationTarget
			var organizationID, ownerOrganizationIDActual, externalIdentifier string
			if err := tx.QueryRow(ctx, `SELECT id,target_kind,COALESCE(organization_id,''),COALESCE(owner_organization_id,''),COALESCE(external_identifier,'') FROM regulated_targets WHERE id=$1`, target.ID).Scan(&actual.ID, &actual.TargetKind, &organizationID, &ownerOrganizationIDActual, &externalIdentifier); err != nil {
				return fmt.Errorf("verify foundation regulated target %s: %w", target.ID, err)
			}
			actual.OrganizationID = organizationID
			actual.OwnerOrganizationID = ownerOrganizationIDActual
			if externalIdentifier != "" {
				actual.ExternalIdentifier = &externalIdentifier
			}
			if actual.ID != target.ID || actual.TargetKind != target.TargetKind || actual.OrganizationID != target.OrganizationID || actual.OwnerOrganizationID != target.OwnerOrganizationID || !sameOptionalString(actual.ExternalIdentifier, target.ExternalIdentifier) {
				return fmt.Errorf("foundation regulated target %s drifted", target.ID)
			}
			if ownerOrganizationID == "" {
				return fmt.Errorf("foundation regulated target %s has no owner", target.ID)
			}
		}
		for _, declaredScope := range foundationScopes(manifest) {
			if _, err := tx.Exec(ctx, `INSERT INTO organization_service_provider_scopes (id,organization_id,service_provider_type_id,authorization_identifier,certificate_identifier,status,effective_from,effective_to,operation_qualifiers,activity_qualifiers,primary_target_id,created_at) VALUES ($1,$2,$3,$4,NULL,$5,CURRENT_DATE,NULL,'{}'::jsonb,'{}'::jsonb,$6,$7) ON CONFLICT (id) DO NOTHING`, declaredScope.ID, declaredScope.OrganizationID, declaredScope.ServiceProviderTypeID, declaredScope.AuthorizationIdentifier, declaredScope.Status, declaredScope.PrimaryTargetID, now); err != nil {
				return fmt.Errorf("insert foundation provider scope %s: %w", declaredScope.ID, err)
			}
			var scope FoundationScope
			if err := tx.QueryRow(ctx, `SELECT id,organization_id,service_provider_type_id,authorization_identifier,status,COALESCE(primary_target_id,'') FROM organization_service_provider_scopes WHERE id=$1`, declaredScope.ID).Scan(&scope.ID, &scope.OrganizationID, &scope.ServiceProviderTypeID, &scope.AuthorizationIdentifier, &scope.Status, &scope.PrimaryTargetID); err != nil {
				return fmt.Errorf("verify foundation provider scope %s: %w", declaredScope.ID, err)
			}
			if scope.ID != declaredScope.ID || scope.OrganizationID != declaredScope.OrganizationID || scope.ServiceProviderTypeID != declaredScope.ServiceProviderTypeID || scope.AuthorizationIdentifier != declaredScope.AuthorizationIdentifier || scope.Status != declaredScope.Status || scope.PrimaryTargetID != declaredScope.PrimaryTargetID {
				return fmt.Errorf("foundation provider scope %s drifted", declaredScope.ID)
			}
			var providerTypeExists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM service_provider_types WHERE id=$1 AND target_kinds @> ARRAY(SELECT target_kind FROM regulated_targets WHERE id=$2) AND normalization_status='NORMALIZED')`, declaredScope.ServiceProviderTypeID, declaredScope.PrimaryTargetID).Scan(&providerTypeExists); err != nil || !providerTypeExists {
				return fmt.Errorf("foundation provider type is unavailable for scope %s", declaredScope.ID)
			}
			for _, targetID := range foundationScopeTargetIDs(declaredScope) {
				if _, err := tx.Exec(ctx, `INSERT INTO organization_service_provider_scope_targets (organization_service_provider_scope_id,regulated_target_id,created_at) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, declaredScope.ID, targetID, now); err != nil {
					return fmt.Errorf("insert foundation scope target %s/%s: %w", declaredScope.ID, targetID, err)
				}
				var linked bool
				if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM organization_service_provider_scope_targets WHERE organization_service_provider_scope_id=$1 AND regulated_target_id=$2)`, declaredScope.ID, targetID).Scan(&linked); err != nil || !linked {
					return fmt.Errorf("verify foundation scope target %s/%s", declaredScope.ID, targetID)
				}
			}
		}
		var controlScopes, controlTargets int
		if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM organization_service_provider_scopes WHERE organization_id=$1),(SELECT count(*) FROM regulated_targets WHERE organization_id=$1)`, manifest.ControlOrganization.ID).Scan(&controlScopes, &controlTargets); err != nil {
			return err
		}
		if manifest.ControlMustHaveNoProviderScope && controlScopes != 0 {
			return fmt.Errorf("control organization has an unexpected provider scope")
		}
		if manifest.ControlMustHaveNoRegulatedTarget && controlTargets != 0 {
			return fmt.Errorf("control organization has an unexpected regulated target")
		}
		return nil
	})
}

func foundationOrganizations(manifest FoundationManifest) []FoundationOrganization {
	organizations := append([]FoundationOrganization{manifest.TargetOrganization}, manifest.AdditionalTargetOrganizations...)
	return append(organizations, manifest.ControlOrganization)
}

func foundationTargets(manifest FoundationManifest) []FoundationTarget {
	return append([]FoundationTarget{manifest.RegulatedTarget}, manifest.AdditionalRegulatedTargets...)
}

func foundationScopes(manifest FoundationManifest) []FoundationScope {
	return append([]FoundationScope{manifest.ProviderScope}, manifest.AdditionalProviderScopes...)
}

func foundationScopeTargetIDs(scope FoundationScope) []string {
	if len(scope.TargetIDs) > 0 {
		return append([]string(nil), scope.TargetIDs...)
	}
	return []string{scope.PrimaryTargetID}
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableOptionalString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func sameOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func ensureBootstrapActor(ctx context.Context, tx pgx.Tx, subjectID string, now time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO identity_references (subject_id,issuer,display_name,revision,email,created_at) VALUES ($1,'avia:bootstrap','Avia deployment bootstrap',1,NULL,$2) ON CONFLICT (subject_id) DO NOTHING`, subjectID, now)
	if err != nil {
		return fmt.Errorf("ensure deployment bootstrap actor: %w", err)
	}
	var issuer, displayName string
	if err := tx.QueryRow(ctx, `SELECT issuer,display_name FROM identity_references WHERE subject_id=$1`, subjectID).Scan(&issuer, &displayName); err != nil {
		return fmt.Errorf("verify deployment bootstrap actor: %w", err)
	}
	if issuer != "avia:bootstrap" || displayName != "Avia deployment bootstrap" {
		return fmt.Errorf("deployment bootstrap actor is not a non-login service identity")
	}
	return nil
}
