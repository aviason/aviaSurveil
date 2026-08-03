package agacandidatedemo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

// PostgresReader accesses only sealed views with the dedicated reader pool.
// It has no mutation method and never opens the writer/normal API pool.
type PostgresReader struct{ pool *database.Pool }

func NewPostgresReader(pool *database.Pool) (*PostgresReader, error) {
	if pool == nil {
		return nil, fmt.Errorf("AGA candidate demo reader pool is required")
	}
	return &PostgresReader{pool: pool}, nil
}

func (reader *PostgresReader) Capability(ctx context.Context) (Capability, error) {
	var count int
	if err := reader.pool.QueryRow(ctx, `SELECT COUNT(*) FROM preprod_aga_demo.sealed_packages`).Scan(&count); err != nil {
		return Capability{}, ErrUnavailable
	}
	if count != 1 {
		return Capability{}, ErrUnavailable
	}
	return Capability{Available: true, Labels: fixedLabels()}, nil
}

func (reader *PostgresReader) Summary(ctx context.Context) (Summary, error) {
	var summary Summary
	var requirements []byte
	if err := reader.pool.QueryRow(ctx, `SELECT p.package_digest, (SELECT COUNT(*) FROM preprod_aga_demo.sealed_forms), (SELECT COUNT(*) FROM preprod_aga_demo.sealed_questions), p.payload->'sourceResolutionRequirements' FROM preprod_aga_demo.sealed_packages p`).Scan(&summary.PackageDigest, &summary.FormCount, &summary.QuestionCount, &requirements); err != nil {
		return Summary{}, ErrNotFound
	}
	if summary.FormCount != 52 || summary.QuestionCount != 1310 {
		return Summary{}, ErrUnavailable
	}
	if err := json.Unmarshal(requirements, &summary.SourceRequirements); err != nil || !exactRequirements(summary.SourceRequirements) {
		return Summary{}, ErrUnavailable
	}
	summary.Labels = fixedLabels()
	return summary, nil
}

func exactRequirements(values []string) bool {
	expected := []string{"EXACT_SOURCE_BYTES", "EXACT_SOURCE_BYTES_SHA256", "EFFECTIVE_DATE", "CLAUSE_OR_PAGE_LOCATOR", "APPLICABILITY", "NAMED_SOURCE_OWNER_ATTESTATION"}
	if len(values) != len(expected) {
		return false
	}
	for index, value := range expected {
		if values[index] != value {
			return false
		}
	}
	return true
}

func (reader *PostgresReader) Forms(ctx context.Context, cursor string, limit int) (Page[Form], error) {
	rows, err := reader.pool.Query(ctx, `SELECT form_code, payload->'form'->>'documentTitle', COALESCE((payload->'form'->>'questionCount')::integer, 0), payload->'form'->>'questionExtractionState' FROM preprod_aga_demo.sealed_forms WHERE form_code > $1 ORDER BY form_code LIMIT $2`, cursor, limit+1)
	if err != nil {
		return Page[Form]{}, ErrNotFound
	}
	defer rows.Close()
	page := Page[Form]{Items: make([]Form, 0, limit)}
	for rows.Next() {
		var item Form
		if err := rows.Scan(&item.Code, &item.Title, &item.QuestionCount, &item.QuestionExtractionState); err != nil {
			return Page[Form]{}, ErrNotFound
		}
		if len(page.Items) == limit {
			next := page.Items[len(page.Items)-1].Code
			page.NextCursor = &next
			break
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Page[Form]{}, ErrNotFound
	}
	return page, nil
}

func (reader *PostgresReader) Form(ctx context.Context, code string) (Form, error) {
	var item Form
	if err := reader.pool.QueryRow(ctx, `SELECT form_code, payload->'form'->>'documentTitle', COALESCE((payload->'form'->>'questionCount')::integer, 0), payload->'form'->>'questionExtractionState' FROM preprod_aga_demo.sealed_forms WHERE form_code = $1`, code).Scan(&item.Code, &item.Title, &item.QuestionCount, &item.QuestionExtractionState); err != nil {
		return Form{}, ErrNotFound
	}
	return item, nil
}

func (reader *PostgresReader) Questions(ctx context.Context, cursor, formCode, sourceGapCategory, riskBand string, limit int) (Page[Question], error) {
	rows, err := reader.pool.Query(ctx, `SELECT proposal_id, form_code, ordinal, payload->'question'->>'originalText', text_digest, source_gap_category, payload->'question'->'proposedRisk'->>'band' FROM preprod_aga_demo.sealed_questions WHERE proposal_id > $1 AND ($2 = '' OR form_code = $2) AND ($3 = '' OR source_gap_category = $3) AND ($4 = '' OR payload->'question'->'proposedRisk'->>'band' = $4) ORDER BY proposal_id LIMIT $5`, cursor, formCode, sourceGapCategory, riskBand, limit+1)
	if err != nil {
		return Page[Question]{}, ErrNotFound
	}
	defer rows.Close()
	page := Page[Question]{Items: make([]Question, 0, limit)}
	for rows.Next() {
		var item Question
		if err := rows.Scan(&item.ProposalID, &item.FormCode, &item.Ordinal, &item.Text, &item.TextDigest, &item.SourceGapCategory, &item.RiskBand); err != nil {
			return Page[Question]{}, ErrNotFound
		}
		if len(page.Items) == limit {
			next := page.Items[len(page.Items)-1].ProposalID
			page.NextCursor = &next
			break
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Page[Question]{}, ErrNotFound
	}
	return page, nil
}

func fixedLabels() []string {
	return []string{"candidate-only", "release pending", "production-ready: not established"}
}
