package configuration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

var (
	ErrWorkspaceForbidden = errors.New("configuration workspace forbidden")
	ErrWorkspaceNotFound  = errors.New("configuration workspace record not found")
	ErrWorkspaceInvalid   = errors.New("invalid configuration workspace request")
)

type AdminQuestion struct {
	ID                  string `json:"id"`
	Prompt              string `json:"prompt"`
	ConfiguredReference string `json:"configuredReference"`
	ExpectedEvidence    string `json:"expectedEvidence"`
	Revision            int64  `json:"revision"`
}

type AdminRegulatorySource struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	SourceType string  `json:"sourceType"`
	Version    string  `json:"version"`
	Status     string  `json:"status"`
	Locator    string  `json:"locator"`
	URL        *string `json:"url"`
}

type AdminProposedInspectionQuestion struct {
	ID                 string   `json:"id"`
	Prompt             string   `json:"prompt"`
	VerificationMethod string   `json:"verificationMethod"`
	EvidenceExamples   []string `json:"evidenceExamples"`
	WhyIncluded        string   `json:"whyIncluded"`
}

type AdminRegulatoryRefreshPolicy struct {
	SourceCollectionID             string   `json:"sourceCollectionId"`
	LastCheckedAt                  string   `json:"lastCheckedAt"`
	NextReconciliationDate         string   `json:"nextReconciliationDate"`
	NextExpertValidationDate       string   `json:"nextExpertValidationDate"`
	EventDrivenReview              bool     `json:"eventDrivenReview"`
	ReconciliationIntervalMonths   int64    `json:"reconciliationIntervalMonths"`
	ExpertValidationIntervalMonths int64    `json:"expertValidationIntervalMonths"`
	SourceChangeState              string   `json:"sourceChangeState"`
	UpdateMode                     string   `json:"updateMode"`
	DocumentCount                  int64    `json:"documentCount"`
	ManifestPath                   string   `json:"manifestPath"`
	Guardrails                     []string `json:"guardrails"`
}

type AdminChecklistQuestionScopeRecommendation struct {
	QuestionID              string `json:"questionId"`
	Classification          string `json:"classification"`
	Rationale               string `json:"rationale"`
	HistoryBasis            string `json:"historyBasis"`
	RequiresManagerApproval bool   `json:"requiresManagerApproval"`
}

type AdminChecklistScopeRecommendation struct {
	ID                      string                                      `json:"id"`
	Status                  string                                      `json:"status"`
	HistoryState            string                                      `json:"historyState"`
	GeneratedAt             string                                      `json:"generatedAt"`
	Signals                 []string                                    `json:"signals"`
	Guardrails              []string                                    `json:"guardrails"`
	QuestionRecommendations []AdminChecklistQuestionScopeRecommendation `json:"questionRecommendations"`
}

type AdminRegulatoryMapping struct {
	ID                         string                            `json:"id"`
	AuditArea                  string                            `json:"auditArea"`
	ServiceProviderTypes       []string                          `json:"serviceProviderTypes"`
	ApplicableRegulations      []string                          `json:"applicableRegulations"`
	CriticalElement            string                            `json:"criticalElement"`
	ProtocolQuestionID         string                            `json:"protocolQuestionId"`
	ProtocolQuestion           string                            `json:"protocolQuestion"`
	AnnexReferences            []string                          `json:"annexReferences"`
	NationalReferences         []string                          `json:"nationalReferences"`
	CAAImplementationReference string                            `json:"caaImplementationReference"`
	Requirement                string                            `json:"requirement"`
	VerificationObjective      string                            `json:"verificationObjective"`
	ExpectedEvidence           []string                          `json:"expectedEvidence"`
	WhyIncluded                string                            `json:"whyIncluded"`
	ReviewStatus               string                            `json:"reviewStatus"`
	SourceGap                  *string                           `json:"sourceGap"`
	RefreshPolicy              AdminRegulatoryRefreshPolicy      `json:"refreshPolicy"`
	ScopeRecommendation        AdminChecklistScopeRecommendation `json:"scopeRecommendation"`
	Sources                    []AdminRegulatorySource           `json:"sources"`
	ProposedQuestions          []AdminProposedInspectionQuestion `json:"proposedQuestions"`
}

type AdminRegulatoryReference struct {
	ID              string                   `json:"id"`
	Title           string                   `json:"title"`
	Version         string                   `json:"version"`
	Status          string                   `json:"status"`
	EffectiveDate   string                   `json:"effectiveDate"`
	ConfiguredRules []string                 `json:"configuredRules"`
	ChangeHistory   []string                 `json:"changeHistory"`
	Mappings        []AdminRegulatoryMapping `json:"mappings"`
}

type AdminTemplateMaster struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	PublishedVersionID string  `json:"publishedVersionId"`
	Status             string  `json:"status"`
	Owner              string  `json:"owner"`
	ItemCount          int64   `json:"itemCount"`
	PreviewPath        *string `json:"previewPath"`
	DisabledReason     *string `json:"disabledReason"`
	Revision           int64   `json:"revision"`
}

type AdminTemplateVersion struct {
	ID               string    `json:"id"`
	TemplateID       string    `json:"templateId"`
	Version          int64     `json:"version"`
	Status           string    `json:"status"`
	Owner            string    `json:"owner"`
	CreatorSubjectID string    `json:"creatorSubjectId"`
	ChangeReason     string    `json:"changeReason"`
	QuestionIDs      []string  `json:"questionIds"`
	Revision         int64     `json:"revision"`
	CreatedAt        time.Time `json:"createdAt"`
}

type AdminTemplate struct {
	ID                 string                 `json:"id"`
	PublishedVersionID string                 `json:"publishedVersionId"`
	Versions           []AdminTemplateVersion `json:"versions"`
	Revision           int64                  `json:"revision"`
}

type AdminInspectionPackage struct {
	ID                   string   `json:"id"`
	AuditID              string   `json:"auditId"`
	OrganizationID       string   `json:"organizationId"`
	OrganizationName     string   `json:"organizationName"`
	QuestionIDs          []string `json:"questionIds"`
	ConfiguredReferences []string `json:"configuredReferences"`
	ExpectedEvidence     []string `json:"expectedEvidence"`
	RiskFocus            []string `json:"riskFocus"`
}

type WorkspaceService struct {
	pool *database.Pool
}

func NewWorkspaceService(pool *database.Pool) *WorkspaceService {
	return &WorkspaceService{pool: pool}
}

func (service *WorkspaceService) ListRegulatoryReferences(
	ctx context.Context,
	actor identity.Principal,
	search string,
	status string,
	limit int,
) ([]AdminRegulatoryReference, error) {
	if !CanPreview(actor) {
		return nil, fmt.Errorf("%w: Admin Preview authority is required", ErrWorkspaceForbidden)
	}
	if service == nil || service.pool == nil {
		return nil, ErrWorkspaceInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := service.pool.Query(ctx, `
		SELECT reference_id, title,
			COALESCE(snapshot->>'versionLabel', version::text),
			status, effective_date::text, snapshot
		FROM (
			SELECT DISTINCT ON (reference_id) *
			FROM regulatory_reference_versions
			ORDER BY reference_id, version DESC
		) latest
		WHERE ($1 = '' OR concat_ws(' ', reference_id, title, snapshot->>'versionLabel')
			ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR status = $2)
		ORDER BY reference_id
		LIMIT $3
	`, strings.TrimSpace(search), strings.TrimSpace(status), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminRegulatoryReference, 0)
	for rows.Next() {
		var item AdminRegulatoryReference
		var snapshot []byte
		if err := rows.Scan(
			&item.ID, &item.Title, &item.Version, &item.Status,
			&item.EffectiveDate, &snapshot,
		); err != nil {
			return nil, err
		}
		var detail struct {
			ConfiguredRules []string                 `json:"configuredRules"`
			ChangeHistory   []string                 `json:"changeHistory"`
			Mappings        []AdminRegulatoryMapping `json:"mappings"`
		}
		if err := json.Unmarshal(snapshot, &detail); err != nil {
			return nil, fmt.Errorf("decode regulatory reference snapshot: %w", err)
		}
		item.ConfiguredRules = detail.ConfiguredRules
		item.ChangeHistory = detail.ChangeHistory
		item.Mappings = detail.Mappings
		if item.ConfiguredRules == nil {
			item.ConfiguredRules = []string{}
		}
		if item.ChangeHistory == nil {
			item.ChangeHistory = []string{}
		}
		if item.Mappings == nil {
			item.Mappings = []AdminRegulatoryMapping{}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (service *WorkspaceService) ListQuestions(
	ctx context.Context,
	actor identity.Principal,
	search string,
	limit int,
) ([]AdminQuestion, error) {
	if !CanPreview(actor) {
		return nil, fmt.Errorf("%w: Admin Preview authority is required", ErrWorkspaceForbidden)
	}
	if service == nil || service.pool == nil {
		return nil, ErrWorkspaceInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := service.pool.Query(ctx, `
		SELECT question_id, prompt, configured_reference, expected_evidence, version::bigint
		FROM (
			SELECT DISTINCT ON (question_id)
				question_id, prompt, configured_reference, expected_evidence, version
			FROM question_versions
			ORDER BY question_id, version DESC
		) latest
		WHERE $1 = ''
		   OR concat_ws(' ', question_id, prompt, configured_reference, expected_evidence)
				ILIKE '%' || $1 || '%'
		ORDER BY question_id
		LIMIT $2
	`, strings.TrimSpace(search), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminQuestion, 0)
	for rows.Next() {
		var item AdminQuestion
		if err := rows.Scan(
			&item.ID, &item.Prompt, &item.ConfiguredReference,
			&item.ExpectedEvidence, &item.Revision,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (service *WorkspaceService) ListTemplateMasters(
	ctx context.Context,
	actor identity.Principal,
	limit int,
) ([]AdminTemplateMaster, error) {
	if !CanPreview(actor) {
		return nil, fmt.Errorf("%w: Admin Preview authority is required", ErrWorkspaceForbidden)
	}
	if service == nil || service.pool == nil {
		return nil, ErrWorkspaceInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := service.pool.Query(ctx, `
		SELECT master.id, master.title, master.published_template_version_id,
			master.owner_role, master.revision,
			count(link.question_version_id)::bigint
		FROM template_masters master
		LEFT JOIN template_version_questions link
		  ON link.template_version_id = master.published_template_version_id
		WHERE master.tombstoned_at IS NULL
		GROUP BY master.id, master.title, master.published_template_version_id,
			master.owner_role, master.revision
		ORDER BY master.id
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminTemplateMaster, 0)
	for rows.Next() {
		var item AdminTemplateMaster
		if err := rows.Scan(
			&item.ID, &item.Title, &item.PublishedVersionID,
			&item.Owner, &item.Revision, &item.ItemCount,
		); err != nil {
			return nil, err
		}
		item.Status = "PUBLISHED"
		if item.ID == "TPL-CABIN-2026" {
			previewPath := "/admin/templates"
			item.PreviewPath = &previewPath
		} else {
			reason := item.ID + " / " + item.PublishedVersionID +
				" has no declared Template Preview route in Task 10."
			item.DisabledReason = &reason
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (service *WorkspaceService) GetTemplate(
	ctx context.Context,
	actor identity.Principal,
	templateID string,
) (AdminTemplate, error) {
	if !CanPreview(actor) {
		return AdminTemplate{}, fmt.Errorf("%w: Admin Preview authority is required", ErrWorkspaceForbidden)
	}
	if service == nil || service.pool == nil || strings.TrimSpace(templateID) == "" {
		return AdminTemplate{}, ErrWorkspaceInvalid
	}
	var output AdminTemplate
	if err := service.pool.QueryRow(ctx, `
		SELECT id, published_template_version_id, revision
		FROM template_masters
		WHERE id = $1 AND tombstoned_at IS NULL
	`, templateID).Scan(&output.ID, &output.PublishedVersionID, &output.Revision); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminTemplate{}, ErrWorkspaceNotFound
		}
		return AdminTemplate{}, err
	}

	var published AdminTemplateVersion
	if err := service.pool.QueryRow(ctx, `
		SELECT version.id, version.version::bigint,
			COALESCE(version.snapshot->>'creatorSubjectId', 'USR-MANAGER-NORA'),
			COALESCE(version.snapshot->>'changeReason', 'Initial immutable published version.'),
			COALESCE(
				array_agg(question.question_id ORDER BY link.position)
					FILTER (WHERE question.question_id IS NOT NULL),
				ARRAY[]::text[]
			),
			version.published_at
		FROM checklist_template_versions version
		LEFT JOIN template_version_questions link ON link.template_version_id = version.id
		LEFT JOIN question_versions question ON question.id = link.question_version_id
		WHERE version.id = $1
		GROUP BY version.id, version.version, version.snapshot, version.published_at
	`, output.PublishedVersionID).Scan(
		&published.ID, &published.Version, &published.CreatorSubjectID,
		&published.ChangeReason, &published.QuestionIDs, &published.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminTemplate{}, ErrWorkspaceNotFound
		}
		return AdminTemplate{}, err
	}
	published.TemplateID = output.ID
	published.Status = "PUBLISHED"
	published.Owner = "Department Manager"
	published.Revision = 1
	output.Versions = []AdminTemplateVersion{published}

	rows, err := service.pool.Query(ctx, `
		SELECT id, version::bigint, owner_role, creator_subject_id,
			change_reason, question_version_ids, revision, created_at
		FROM template_draft_versions
		WHERE template_id = $1 AND status = 'DRAFT'
		ORDER BY version, revision
	`, templateID)
	if err != nil {
		return AdminTemplate{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var draft AdminTemplateVersion
		var questionVersionIDs []string
		if err := rows.Scan(
			&draft.ID, &draft.Version, &draft.Owner, &draft.CreatorSubjectID,
			&draft.ChangeReason, &questionVersionIDs, &draft.Revision, &draft.CreatedAt,
		); err != nil {
			return AdminTemplate{}, err
		}
		draft.TemplateID = output.ID
		draft.Status = "DRAFT"
		draft.QuestionIDs, err = service.questionIDs(ctx, questionVersionIDs)
		if err != nil {
			return AdminTemplate{}, err
		}
		output.Versions = append(output.Versions, draft)
	}
	if err := rows.Err(); err != nil {
		return AdminTemplate{}, err
	}
	return output, nil
}

func (service *WorkspaceService) GetInspectionPackage(
	ctx context.Context,
	actor identity.Principal,
	packageID string,
) (AdminInspectionPackage, error) {
	if !CanPreview(actor) {
		return AdminInspectionPackage{}, fmt.Errorf(
			"%w: Admin Preview authority is required",
			ErrWorkspaceForbidden,
		)
	}
	if service == nil || service.pool == nil || strings.TrimSpace(packageID) == "" {
		return AdminInspectionPackage{}, ErrWorkspaceInvalid
	}
	var output AdminInspectionPackage
	var snapshot []byte
	if err := service.pool.QueryRow(ctx, `
		SELECT package.id, package.inspection_id, inspection.organization_id,
			organization.legal_name, package.snapshot
		FROM inspection_packages package
		JOIN inspections inspection ON inspection.id = package.inspection_id
		JOIN organizations organization ON organization.id = inspection.organization_id
		WHERE package.id = $1
	`, packageID).Scan(
		&output.ID, &output.AuditID, &output.OrganizationID,
		&output.OrganizationName, &snapshot,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminInspectionPackage{}, ErrWorkspaceNotFound
		}
		return AdminInspectionPackage{}, err
	}
	var packageSnapshot struct {
		RiskFocus []string `json:"riskFocus"`
		Questions []struct {
			ID                  string `json:"id"`
			RegulatoryReference string `json:"regulatoryReference"`
			ExpectedEvidence    string `json:"expectedEvidence"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(snapshot, &packageSnapshot); err != nil {
		return AdminInspectionPackage{}, fmt.Errorf("decode immutable package snapshot: %w", err)
	}
	for _, question := range packageSnapshot.Questions {
		output.QuestionIDs = append(output.QuestionIDs, question.ID)
		output.ConfiguredReferences = append(
			output.ConfiguredReferences,
			question.RegulatoryReference,
		)
		output.ExpectedEvidence = append(output.ExpectedEvidence, question.ExpectedEvidence)
	}
	output.RiskFocus = append([]string{}, packageSnapshot.RiskFocus...)
	return output, nil
}

func (service *WorkspaceService) questionIDs(
	ctx context.Context,
	questionVersionIDs []string,
) ([]string, error) {
	if len(questionVersionIDs) == 0 {
		return []string{}, nil
	}
	rows, err := service.pool.Query(ctx, `
		SELECT question.question_id
		FROM unnest($1::text[]) WITH ORDINALITY AS selected(id, position)
		JOIN question_versions question ON question.id = selected.id
		ORDER BY selected.position
	`, questionVersionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	questionIDs := make([]string, 0, len(questionVersionIDs))
	for rows.Next() {
		var questionID string
		if err := rows.Scan(&questionID); err != nil {
			return nil, err
		}
		questionIDs = append(questionIDs, questionID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(questionIDs) != len(questionVersionIDs) {
		return nil, fmt.Errorf("%w: Draft references an unknown Question version", ErrWorkspaceInvalid)
	}
	return questionIDs, nil
}
