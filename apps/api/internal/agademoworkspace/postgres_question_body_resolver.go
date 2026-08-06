//go:build preproddemo

package agademoworkspace

import (
	"context"
	"fmt"
	"strings"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

// PostgresQuestionBodyResolver owns only a dedicated sealed-overlay reader
// pool. It has no workspace or canonical store capability and resolves every
// row by the complete immutable identity supplied by the service.
type PostgresQuestionBodyResolver struct {
	pool *database.Pool
}

func NewPostgresQuestionBodyResolver(pool *database.Pool) (*PostgresQuestionBodyResolver, error) {
	if pool == nil {
		return nil, fmt.Errorf("AGA question body overlay reader pool is required")
	}
	return &PostgresQuestionBodyResolver{pool: pool}, nil
}

func (resolver *PostgresQuestionBodyResolver) Resolve(ctx context.Context, identities []aga.BaseIdentity) ([]QuestionBody, error) {
	if resolver == nil || resolver.pool == nil || len(identities) > MaxQuestionTextPage {
		return nil, ErrQuestionBodyResolverUnavailable
	}
	if len(identities) == 0 {
		return []QuestionBody{}, nil
	}
	proposalIDs := make([]string, 0, len(identities))
	formCodes := make([]string, 0, len(identities))
	ordinals := make([]int, 0, len(identities))
	digests := make([]string, 0, len(identities))
	for _, identity := range identities {
		proposalIDs = append(proposalIDs, identity.ProposalID)
		formCodes = append(formCodes, identity.FormCode)
		ordinals = append(ordinals, identity.Ordinal)
		digests = append(digests, identity.TextDigest)
	}
	rows, err := resolver.pool.Query(ctx, `
		WITH requested AS (
			SELECT *
			FROM unnest($1::text[], $2::text[], $3::integer[], $4::text[])
				AS r(proposal_id, form_code, ordinal, text_digest)
		)
		SELECT q.proposal_id, q.payload->'question'->>'originalText', q.text_digest, q.form_code, q.ordinal
		FROM preprod_aga_demo.sealed_questions q
		JOIN requested r
		  ON r.proposal_id = q.proposal_id
		 AND r.form_code = q.form_code
		 AND r.ordinal = q.ordinal
		 AND r.text_digest = q.text_digest
	`, proposalIDs, formCodes, ordinals, digests)
	if err != nil {
		return nil, fmt.Errorf("sealed question body lookup failed")
	}
	defer rows.Close()
	byIdentity := make(map[string]QuestionBody, len(identities))
	for rows.Next() {
		var proposalID, text, digest, formCode string
		var ordinal int
		if err := rows.Scan(&proposalID, &text, &digest, &formCode, &ordinal); err != nil {
			return nil, fmt.Errorf("sealed question body lookup failed")
		}
		identity := aga.BaseIdentity{PackageVersion: aga.FrozenPackageVersion, PackageJSONSHA256: aga.FrozenPackageJSONSHA256, FormCode: formCode, ProposalID: proposalID, Ordinal: ordinal, TextDigest: digest}
		byIdentity[identity.Key()] = QuestionBody{Identity: identity, Text: text, TextDigest: digest}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sealed question body lookup failed")
	}
	resolved := make([]QuestionBody, 0, len(identities))
	for _, identity := range identities {
		body, found := byIdentity[identity.Key()]
		if !found {
			return nil, fmt.Errorf("sealed question body lookup was incomplete")
		}
		resolved = append(resolved, body)
	}
	return resolved, nil
}

func (resolver *PostgresQuestionBodyResolver) Search(ctx context.Context, value string) ([]aga.BaseIdentity, error) {
	if resolver == nil || resolver.pool == nil {
		return nil, ErrQuestionBodyResolverUnavailable
	}
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" || len(value) > 256 {
		return nil, ErrMalformedCommand
	}
	rows, err := resolver.pool.Query(ctx, `
		SELECT proposal_id, form_code, ordinal, text_digest
		FROM preprod_aga_demo.sealed_questions
		WHERE lower(payload->'question'->>'originalText') LIKE lower('%' || $1 || '%')
		ORDER BY ordinal, proposal_id
	`, value)
	if err != nil {
		return nil, ErrQuestionBodyResolverUnavailable
	}
	defer rows.Close()
	identities := make([]aga.BaseIdentity, 0)
	for rows.Next() {
		var identity aga.BaseIdentity
		if err := rows.Scan(&identity.ProposalID, &identity.FormCode, &identity.Ordinal, &identity.TextDigest); err != nil {
			return nil, ErrQuestionBodyResolverUnavailable
		}
		identity.PackageVersion = aga.FrozenPackageVersion
		identity.PackageJSONSHA256 = aga.FrozenPackageJSONSHA256
		identities = append(identities, identity)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQuestionBodyResolverUnavailable
	}
	return identities, nil
}
