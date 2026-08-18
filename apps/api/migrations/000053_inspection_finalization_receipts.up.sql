CREATE TABLE IF NOT EXISTS inspection_finalization_receipts (
    receipt_id text PRIMARY KEY,
    inspection_id text NOT NULL UNIQUE REFERENCES inspections(id),
    package_revision bigint NOT NULL CHECK (package_revision > 0),
    server_revision bigint NOT NULL CHECK (server_revision > 0),
    answer_manifest_hash text NOT NULL,
    finding_manifest_hash text NOT NULL,
    attachment_manifest_hash text NOT NULL,
    event_manifest_hash text NOT NULL,
    canonicalization_version text NOT NULL CHECK (canonicalization_version = 'avia-finalization-manifest/v1'),
    server_timestamp timestamptz NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS inspection_finalization_receipts_inspection_idx
    ON inspection_finalization_receipts (inspection_id, package_revision);
