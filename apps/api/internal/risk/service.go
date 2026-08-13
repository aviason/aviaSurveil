package risk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	riskstore "github.com/aviason/aviaSurveil/internal/risk/store/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrForbidden = errors.New("governed risk projection authority required")
	ErrInvalid   = errors.New("invalid risk projection request")
	ErrNotFound  = errors.New("risk projection source not found")
)

const (
	overviewKind          = "OVERVIEW"
	managementKind        = "MANAGEMENT"
	overviewSource        = "canonical finding lifecycle projection"
	managementSource      = "canonical finding and CAP lifecycle projection"
	NonDecisionLabel      = "Advisory management information — not an enforcement, certificate, or closure decision."
	ManagementRiskHigh    = ManagementRiskLevel("HIGH")
	ManagementRiskMedium  = ManagementRiskLevel("MEDIUM")
	ManagementRiskLow     = ManagementRiskLevel("LOW")
	ManagementRiskVeryLow = ManagementRiskLevel("VERY_LOW")

	CAPEffectivenessNotEligible = CAPEffectivenessState("NOT_ELIGIBLE")
	CAPEffectivenessPending     = CAPEffectivenessState("PENDING_POST_CLOSURE_VERIFICATION")
)

type ManagementRiskLevel string
type CAPEffectivenessState string

type Overview struct {
	OrganizationID      string    `json:"organizationId"`
	OverdueFindingCount int       `json:"overdueFindingCount"`
	OpenFindingCount    int       `json:"openFindingCount"`
	RepeatFindingCount  int       `json:"repeatFindingCount"`
	Revision            int       `json:"revision"`
	Source              string    `json:"source"`
	CalculatedAt        time.Time `json:"calculatedAt"`
	AdvisoryOnly        bool      `json:"advisoryOnly"`
	NonDecisionLabel    string    `json:"nonDecisionLabel"`
}

type FindingProjection struct {
	FindingID        string              `json:"findingId"`
	FindingNumber    string              `json:"findingNumber"`
	OrganizationID   string              `json:"organizationId"`
	OrganizationName string              `json:"organizationName"`
	InspectionID     string              `json:"inspectionId"`
	InspectionTitle  string              `json:"inspectionTitle,omitempty"`
	Department       string              `json:"department,omitempty"`
	Title            string              `json:"title"`
	Severity         string              `json:"severity"`
	RiskLevel        ManagementRiskLevel `json:"riskLevel"`
	Status           string              `json:"status"`
	IssuedAt         *time.Time          `json:"issuedAt"`
	DueState         string              `json:"dueState"`
	CAPRequired      bool                `json:"capRequired"`
}

type CAPEffectivenessProjection struct {
	FindingID        string                `json:"findingId"`
	FindingNumber    string                `json:"findingNumber"`
	OrganizationID   string                `json:"organizationId"`
	OrganizationName string                `json:"organizationName"`
	FindingStatus    string                `json:"findingStatus"`
	ClosureBasis     string                `json:"closureBasis,omitempty"`
	CAPID            string                `json:"capId"`
	CAPRevisionID    string                `json:"capRevisionId"`
	CAPRevision      int                   `json:"capRevision"`
	CAPStatus        string                `json:"capStatus"`
	State            CAPEffectivenessState `json:"state"`
	Reason           string                `json:"reason"`
}

type ManagementProjection struct {
	Findings         []FindingProjection          `json:"findings"`
	CAPEffectiveness []CAPEffectivenessProjection `json:"capEffectiveness"`
	GeneratedAt      time.Time                    `json:"generatedAt"`
	Revision         int                          `json:"revision"`
	Source           string                       `json:"source"`
	CalculatedAt     time.Time                    `json:"calculatedAt"`
	AdvisoryOnly     bool                         `json:"advisoryOnly"`
	NonDecisionLabel string                       `json:"nonDecisionLabel"`
}

type Dependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
}

type Service struct {
	pool        *database.Pool
	clock       func() time.Time
	idGenerator func(string) string
}

func NewService(pool *database.Pool, dependencies Dependencies) *Service {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = randomID
	}
	return &Service{pool: pool, clock: clock, idGenerator: idGenerator}
}

func (service *Service) GetOverview(
	ctx context.Context,
	actor identity.Principal,
	organizationID string,
) (Overview, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return Overview{}, ErrForbidden
	}
	organizationID = strings.TrimSpace(organizationID)
	var output Overview
	err := database.WithinTransaction(ctx, service.pool, func(
		ctx context.Context,
		transaction pgx.Tx,
	) error {
		lockKey := overviewKind + ":" + organizationID
		if _, err := transaction.Exec(
			ctx,
			"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			"risk-projection:"+lockKey,
		); err != nil {
			return err
		}
		if organizationID != "" {
			var exists bool
			if err := transaction.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM organizations
					WHERE id = $1 AND status = 'ACTIVE' AND tombstoned_at IS NULL
				)
			`, organizationID).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return ErrNotFound
			}
		}
		snapshot := overviewSnapshot{OrganizationID: organizationID}
		if err := transaction.QueryRow(ctx, `
			SELECT
				count(*) FILTER (
					WHERE status <> 'CLOSED'
					  AND due_date IS NOT NULL
					  AND due_date < $1::date
				),
				count(*) FILTER (WHERE status <> 'CLOSED')
			FROM findings
			WHERE tombstoned_at IS NULL
			  AND ($2 = '' OR organization_id = $2)
		`, service.clock().UTC(), organizationID).Scan(
			&snapshot.OverdueFindingCount,
			&snapshot.OpenFindingCount,
		); err != nil {
			return err
		}
		// Repeat Finding classification is intentionally not inferred. It remains
		// zero until a governed repeat-classification source is introduced.
		snapshot.RepeatFindingCount = 0
		record, err := service.persistProjection(
			ctx, transaction, overviewKind, nullableString(organizationID),
			overviewSource, snapshot,
		)
		if err != nil {
			return err
		}
		output = Overview{
			OrganizationID:      snapshot.OrganizationID,
			OverdueFindingCount: snapshot.OverdueFindingCount,
			OpenFindingCount:    snapshot.OpenFindingCount,
			RepeatFindingCount:  snapshot.RepeatFindingCount,
			Revision:            int(record.Version), Source: record.Source,
			CalculatedAt:     record.CalculatedAt.Time.UTC(),
			AdvisoryOnly:     record.AdvisoryOnly,
			NonDecisionLabel: NonDecisionLabel,
		}
		return nil
	})
	return output, err
}

func (service *Service) GetManagementProjection(
	ctx context.Context,
	actor identity.Principal,
) (ManagementProjection, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return ManagementProjection{}, ErrForbidden
	}
	var output ManagementProjection
	err := database.WithinTransaction(ctx, service.pool, func(
		ctx context.Context,
		transaction pgx.Tx,
	) error {
		if _, err := transaction.Exec(
			ctx,
			"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			"risk-projection:"+managementKind,
		); err != nil {
			return err
		}
		rows, err := transaction.Query(ctx, `
			SELECT
				finding.id,
				finding.reference,
				finding.organization_id,
				organization.legal_name,
				finding.inspection_id,
				inspection.title,
				COALESCE(potential.title, finding.reference),
				finding.severity,
				finding.status,
				finding.issued_at,
				finding.due_date,
				finding.cap_required,
				COALESCE(finding.closure_basis, ''),
				COALESCE(cap.id, ''),
				COALESCE(cap.cap_id, ''),
				COALESCE(cap.revision, 0),
				COALESCE(cap.status, '')
			FROM findings finding
			JOIN organizations organization ON organization.id = finding.organization_id
			JOIN inspections inspection ON inspection.id = finding.inspection_id
			LEFT JOIN potential_findings potential
			  ON potential.id = finding.potential_finding_id
			LEFT JOIN LATERAL (
				SELECT revision.id, revision.cap_id, revision.revision, revision.status
				FROM cap_revisions revision
				WHERE revision.finding_id = finding.id
				ORDER BY revision.revision DESC
				LIMIT 1
			) cap ON true
			WHERE finding.tombstoned_at IS NULL
			ORDER BY finding.id
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		snapshot := managementSnapshot{
			Findings:         []FindingProjection{},
			CAPEffectiveness: []CAPEffectivenessProjection{},
		}
		now := service.clock().UTC()
		for rows.Next() {
			var finding FindingProjection
			var dueDate pgtype.Date
			var issuedAt pgtype.Timestamptz
			var closureBasis, capRevisionID, capID, capStatus string
			var capRevision int
			if err := rows.Scan(
				&finding.FindingID,
				&finding.FindingNumber,
				&finding.OrganizationID,
				&finding.OrganizationName,
				&finding.InspectionID,
				&finding.InspectionTitle,
				&finding.Title,
				&finding.Severity,
				&finding.Status,
				&issuedAt,
				&dueDate,
				&finding.CAPRequired,
				&closureBasis,
				&capRevisionID,
				&capID,
				&capRevision,
				&capStatus,
			); err != nil {
				return err
			}
			if issuedAt.Valid {
				value := issuedAt.Time.UTC()
				finding.IssuedAt = &value
			}
			finding.RiskLevel = managementRiskLevel(finding.Severity)
			finding.DueState = dueState(now, dueDate, finding.Status)
			snapshot.Findings = append(snapshot.Findings, finding)
			if capRevisionID != "" {
				effectiveness := CAPEffectivenessProjection{
					FindingID: finding.FindingID, FindingNumber: finding.FindingNumber,
					OrganizationID:   finding.OrganizationID,
					OrganizationName: finding.OrganizationName,
					FindingStatus:    finding.Status, ClosureBasis: closureBasis,
					CAPID: capID, CAPRevisionID: capRevisionID,
					CAPRevision: capRevision, CAPStatus: capStatus,
				}
				if finding.Status == "CLOSED" && closureBasis != "" {
					effectiveness.State = CAPEffectivenessPending
					effectiveness.Reason = fmt.Sprintf(
						"Finding %s closed with %s; no typed post-closure effectiveness verification record is available.",
						finding.FindingID, closureBasis,
					)
				} else {
					effectiveness.State = CAPEffectivenessNotEligible
					effectiveness.Reason = fmt.Sprintf(
						"Finding %s is %s; effectiveness requires Finding closure with a typed closure or verification basis.",
						finding.FindingID, finding.Status,
					)
				}
				snapshot.CAPEffectiveness = append(
					snapshot.CAPEffectiveness, effectiveness,
				)
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		record, err := service.persistProjection(
			ctx, transaction, managementKind, nil, managementSource, snapshot,
		)
		if err != nil {
			return err
		}
		output = ManagementProjection{
			Findings:         snapshot.Findings,
			CAPEffectiveness: snapshot.CAPEffectiveness,
			GeneratedAt:      record.CalculatedAt.Time.UTC(),
			Revision:         int(record.Version), Source: record.Source,
			CalculatedAt:     record.CalculatedAt.Time.UTC(),
			AdvisoryOnly:     record.AdvisoryOnly,
			NonDecisionLabel: NonDecisionLabel,
		}
		return nil
	})
	return output, err
}

type overviewSnapshot struct {
	OrganizationID      string `json:"organizationId"`
	OverdueFindingCount int    `json:"overdueFindingCount"`
	OpenFindingCount    int    `json:"openFindingCount"`
	RepeatFindingCount  int    `json:"repeatFindingCount"`
}

type managementSnapshot struct {
	Findings         []FindingProjection          `json:"findings"`
	CAPEffectiveness []CAPEffectivenessProjection `json:"capEffectiveness"`
}

func (service *Service) persistProjection(
	ctx context.Context,
	transaction pgx.Tx,
	kind string,
	organizationID *string,
	source string,
	snapshot any,
) (riskstore.RiskProjectionVersion, error) {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return riskstore.RiskProjectionVersion{}, err
	}
	store := riskstore.New(transaction)
	versions, err := store.ListRiskProjectionVersions(
		ctx,
		riskstore.ListRiskProjectionVersionsParams{
			ProjectionKind: kind, OrganizationID: organizationID, ResultLimit: 1,
		},
	)
	if err != nil {
		return riskstore.RiskProjectionVersion{}, err
	}
	if len(versions) == 1 &&
		versions[0].Source == source &&
		sameJSON(versions[0].Snapshot, snapshotJSON) {
		return versions[0], nil
	}
	version := int32(1)
	if len(versions) == 1 {
		version = versions[0].Version + 1
	}
	now := service.clock().UTC()
	record, err := store.CreateRiskProjectionVersion(
		ctx,
		riskstore.CreateRiskProjectionVersionParams{
			ID: service.idGenerator("risk-projection"), ProjectionKind: kind,
			OrganizationID: organizationID, Version: version, Source: source,
			Snapshot:     snapshotJSON,
			CalculatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		},
	)
	if err != nil {
		return riskstore.RiskProjectionVersion{}, fmt.Errorf(
			"persist immutable %s risk projection: %w", kind, err,
		)
	}
	return record, nil
}

func sameJSON(left, right []byte) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil ||
		json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func managementRiskLevel(severity string) ManagementRiskLevel {
	switch severity {
	case "LEVEL_1_CRITICAL":
		return ManagementRiskHigh
	case "LEVEL_2_MAJOR":
		return ManagementRiskMedium
	case "LEVEL_3_MINOR":
		return ManagementRiskLow
	default:
		return ManagementRiskVeryLow
	}
}

func dueState(now time.Time, dueDate pgtype.Date, status string) string {
	if status == "CLOSED" || !dueDate.Valid {
		return "NONE"
	}
	today := now.UTC().Truncate(24 * time.Hour)
	due := dueDate.Time.UTC().Truncate(24 * time.Hour)
	days := int(due.Sub(today).Hours() / 24)
	switch {
	case days < 0:
		return "OVERDUE"
	case days == 0:
		return "DUE_TODAY"
	case days <= 7:
		return "DUE_SOON"
	default:
		return "NOT_DUE"
	}
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func randomID(prefix string) string {
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(fmt.Sprintf("generate %s ID: %v", prefix, err))
	}
	return prefix + "-" + hex.EncodeToString(value[:])
}
