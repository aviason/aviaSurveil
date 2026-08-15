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
		for _, organization := range []FoundationOrganization{manifest.TargetOrganization, manifest.ControlOrganization} {
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
		if _, err := tx.Exec(ctx, `INSERT INTO regulated_targets (id,target_kind,organization_id,person_subject_id,owner_organization_id,external_identifier,created_at) VALUES ($1,$2,$3,NULL,NULL,NULL,$4) ON CONFLICT (id) DO NOTHING`, manifest.RegulatedTarget.ID, manifest.RegulatedTarget.TargetKind, manifest.RegulatedTarget.OrganizationID, now); err != nil {
			return fmt.Errorf("insert foundation regulated target: %w", err)
		}
		var targetKind, targetOrganization string
		if err := tx.QueryRow(ctx, `SELECT target_kind,COALESCE(organization_id,'') FROM regulated_targets WHERE id=$1`, manifest.RegulatedTarget.ID).Scan(&targetKind, &targetOrganization); err != nil {
			return fmt.Errorf("verify foundation regulated target: %w", err)
		}
		if targetKind != manifest.RegulatedTarget.TargetKind || targetOrganization != manifest.RegulatedTarget.OrganizationID {
			return fmt.Errorf("foundation regulated target drifted")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO organization_service_provider_scopes (id,organization_id,service_provider_type_id,authorization_identifier,certificate_identifier,status,effective_from,effective_to,operation_qualifiers,activity_qualifiers,primary_target_id,created_at) VALUES ($1,$2,$3,$4,NULL,$5,CURRENT_DATE,NULL,'{}'::jsonb,'{}'::jsonb,$6,$7) ON CONFLICT (id) DO NOTHING`, manifest.ProviderScope.ID, manifest.ProviderScope.OrganizationID, manifest.ProviderScope.ServiceProviderTypeID, manifest.ProviderScope.AuthorizationIdentifier, manifest.ProviderScope.Status, manifest.ProviderScope.PrimaryTargetID, now); err != nil {
			return fmt.Errorf("insert foundation provider scope: %w", err)
		}
		var scope FoundationScope
		if err := tx.QueryRow(ctx, `SELECT id,organization_id,service_provider_type_id,authorization_identifier,status,COALESCE(primary_target_id,'') FROM organization_service_provider_scopes WHERE id=$1`, manifest.ProviderScope.ID).Scan(&scope.ID, &scope.OrganizationID, &scope.ServiceProviderTypeID, &scope.AuthorizationIdentifier, &scope.Status, &scope.PrimaryTargetID); err != nil {
			return fmt.Errorf("verify foundation provider scope: %w", err)
		}
		if scope != manifest.ProviderScope {
			return fmt.Errorf("foundation provider scope drifted")
		}
		var providerTypeExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM service_provider_types WHERE id=$1 AND target_kinds @> ARRAY['ORGANIZATION']::text[] AND normalization_status='NORMALIZED')`, manifest.ProviderScope.ServiceProviderTypeID).Scan(&providerTypeExists); err != nil || !providerTypeExists {
			return fmt.Errorf("foundation provider type is unavailable")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO organization_service_provider_scope_targets (organization_service_provider_scope_id,regulated_target_id,created_at) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, manifest.ProviderScope.ID, manifest.RegulatedTarget.ID, now); err != nil {
			return fmt.Errorf("insert foundation scope target: %w", err)
		}
		var linked bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM organization_service_provider_scope_targets WHERE organization_service_provider_scope_id=$1 AND regulated_target_id=$2)`, manifest.ProviderScope.ID, manifest.RegulatedTarget.ID).Scan(&linked); err != nil || !linked {
			return fmt.Errorf("verify foundation scope target")
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
