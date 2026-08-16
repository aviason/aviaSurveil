-- Offline AI advisory enrichment. This table is intentionally separate from
-- the approved source membership so AI interpretation can never rewrite the
-- source question identity or become an approval/publication record.
CREATE TABLE canonical_question_catalog_ai_enrichments (
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    question_version_id text NOT NULL REFERENCES question_versions(id),
    artifact_version text NOT NULL CHECK (btrim(artifact_version) <> ''),
    artifact_digest text NOT NULL CHECK (governed_sha256(artifact_digest)),
    source_catalog_root_digest text NOT NULL CHECK (governed_sha256(source_catalog_root_digest)),
    classification_run_id text NOT NULL CHECK (btrim(classification_run_id) <> ''),
    classification_run_digest text NOT NULL CHECK (governed_sha256(classification_run_digest)),
    prompt_digest text NOT NULL CHECK (governed_sha256(prompt_digest)),
    taxonomy_version text NOT NULL CHECK (btrim(taxonomy_version) <> ''),
    taxonomy_digest text NOT NULL CHECK (governed_sha256(taxonomy_digest)),
    recommendation_policy_version text NOT NULL CHECK (btrim(recommendation_policy_version) <> ''),
    domain_code text NOT NULL CHECK (btrim(domain_code) <> ''),
    topic_codes text[] NOT NULL DEFAULT '{}',
    inspection_type_codes text[] NOT NULL DEFAULT '{}',
    inspection_profile_codes text[] NOT NULL DEFAULT '{}',
    applicability_disposition text NOT NULL CHECK (btrim(applicability_disposition) <> ''),
    risk_band text NOT NULL CHECK (risk_band IN ('PROPOSED_SAFETY_CRITICAL', 'PROPOSED_HIGH_OPERATIONAL', 'PROPOSED_CONTROL_ASSURANCE', 'PROPOSED_REVIEW_REQUIRED')),
    risk_tier text NOT NULL CHECK (risk_tier IN ('HIGH', 'MEDIUM', 'LOW', 'UNKNOWN')),
    safety_critical boolean NOT NULL,
    agreement_confidence text NOT NULL CHECK (agreement_confidence IN ('HIGH', 'MEDIUM', 'LOW')),
    advisory_state text NOT NULL CHECK (advisory_state IN ('SUGGESTED_NOW', 'MATCHING_OPTIONAL', 'UNCERTAIN_SIGNAL', 'RECENTLY_VERIFIED', 'OUTSIDE_FOCUS')),
    default_recommendation_bucket text NOT NULL CHECK (default_recommendation_bucket IN ('SUGGESTED_NOW', 'MATCHING_OPTIONAL', 'UNCERTAIN_SIGNAL')),
    recurrence_months integer NOT NULL CHECK (recurrence_months > 0 AND recurrence_months <= 120),
    rationale_codes text[] NOT NULL DEFAULT '{}',
    external_applicability_unresolved boolean NOT NULL,
    loaded_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (catalog_id, question_version_id),
    FOREIGN KEY (catalog_id, question_version_id)
        REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id)
);

CREATE INDEX canonical_question_catalog_ai_enrichment_domain_idx
    ON canonical_question_catalog_ai_enrichments (catalog_id, domain_code, risk_tier);

CREATE INDEX canonical_question_catalog_ai_enrichment_type_idx
    ON canonical_question_catalog_ai_enrichments (catalog_id, inspection_type_codes);

CREATE INDEX canonical_question_catalog_ai_enrichment_topic_idx
    ON canonical_question_catalog_ai_enrichments (catalog_id, topic_codes);

CREATE TRIGGER canonical_question_catalog_ai_enrichments_append_only
BEFORE UPDATE OR DELETE ON canonical_question_catalog_ai_enrichments
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
