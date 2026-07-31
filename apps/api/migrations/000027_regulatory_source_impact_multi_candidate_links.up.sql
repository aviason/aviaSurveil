-- One controlled source change can affect multiple independent candidate
-- roots. Each candidate/run remains immutable and can bind only once, while a
-- shared impact-review Draft may record all affected candidates.

ALTER TABLE regulatory_source_impact_candidate_links
    DROP CONSTRAINT IF EXISTS regulatory_source_impact_candidate_links_impact_review_draft_id_key;

CREATE INDEX IF NOT EXISTS regulatory_source_impact_candidate_links_draft_idx
    ON regulatory_source_impact_candidate_links (impact_review_draft_id, created_at, id);
