-- Task 6 source currentness is an explicit controlled activation, not an
-- inference from a supplied/observed source row or an effective-date sort.
-- The ledger makes an exact predecessor/current hash pair immutable and
-- creates a source-impact review Draft before any candidate can bind to it.

CREATE TABLE regulatory_source_currentness_events (
    sequence_id bigint GENERATED ALWAYS AS IDENTITY UNIQUE NOT NULL,
    event_id text PRIMARY KEY,
    source_identity text NOT NULL CHECK (btrim(source_identity) <> ''),
    previous_source_version_id text REFERENCES regulatory_source_versions(id),
    previous_source_hash text,
    current_source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    current_source_hash text NOT NULL CHECK (governed_sha256(current_source_hash)),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    operation_id text NOT NULL UNIQUE CHECK (btrim(operation_id) <> ''),
    idempotency_key text NOT NULL UNIQUE CHECK (btrim(idempotency_key) <> ''),
    semantic_payload_digest text NOT NULL CHECK (governed_sha256(semantic_payload_digest)),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    activated_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((previous_source_version_id IS NULL) = (previous_source_hash IS NULL)),
    CHECK (previous_source_hash IS NULL OR governed_sha256(previous_source_hash)),
    CHECK (previous_source_version_id IS NULL OR previous_source_version_id <> current_source_version_id)
);

CREATE UNIQUE INDEX regulatory_source_currentness_events_exact_transition_unique
    ON regulatory_source_currentness_events (
        source_identity,
        COALESCE(previous_source_version_id, ''),
        COALESCE(previous_source_hash, ''),
        current_source_version_id,
        current_source_hash
    );
CREATE INDEX regulatory_source_currentness_events_predecessor_idx
    ON regulatory_source_currentness_events (previous_source_version_id, previous_source_hash, sequence_id);
CREATE INDEX regulatory_source_currentness_events_source_sequence_idx
    ON regulatory_source_currentness_events (source_identity, sequence_id);

CREATE TABLE regulatory_source_impact_review_drafts (
    id text PRIMARY KEY,
    currentness_event_id text NOT NULL UNIQUE REFERENCES regulatory_source_currentness_events(event_id),
    source_identity text NOT NULL CHECK (btrim(source_identity) <> ''),
    previous_source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    previous_source_hash text NOT NULL CHECK (governed_sha256(previous_source_hash)),
    current_source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    current_source_hash text NOT NULL CHECK (governed_sha256(current_source_hash)),
    status text NOT NULL CHECK (status = 'IMPACT_REVIEW_DRAFT'),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (previous_source_version_id <> current_source_version_id)
);

CREATE TABLE regulatory_generation_run_source_currentness_bindings (
    generation_run_id text PRIMARY KEY REFERENCES regulatory_generation_runs(id),
    currentness_event_id text NOT NULL REFERENCES regulatory_source_currentness_events(event_id),
    impact_review_draft_id text REFERENCES regulatory_source_impact_review_drafts(id),
    source_identity text NOT NULL CHECK (btrim(source_identity) <> ''),
    previous_source_version_id text REFERENCES regulatory_source_versions(id),
    previous_source_hash text,
    current_source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    current_source_hash text NOT NULL CHECK (governed_sha256(current_source_hash)),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((previous_source_version_id IS NULL) = (previous_source_hash IS NULL)),
    CHECK (previous_source_hash IS NULL OR governed_sha256(previous_source_hash))
);

CREATE TABLE regulatory_source_impact_candidate_links (
    id text PRIMARY KEY,
    impact_review_draft_id text NOT NULL REFERENCES regulatory_source_impact_review_drafts(id),
    candidate_draft_version_id text NOT NULL UNIQUE REFERENCES template_draft_versions(id),
    generation_run_id text NOT NULL UNIQUE REFERENCES regulatory_generation_runs(id),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX regulatory_source_impact_candidate_links_draft_idx
    ON regulatory_source_impact_candidate_links (impact_review_draft_id, created_at, id);

CREATE TRIGGER regulatory_source_currentness_events_append_only
BEFORE UPDATE OR DELETE ON regulatory_source_currentness_events
FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_source_impact_review_drafts_append_only
BEFORE UPDATE OR DELETE ON regulatory_source_impact_review_drafts
FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_generation_run_source_currentness_bindings_append_only
BEFORE UPDATE OR DELETE ON regulatory_generation_run_source_currentness_bindings
FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_source_impact_candidate_links_append_only
BEFORE UPDATE OR DELETE ON regulatory_source_impact_candidate_links
FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
