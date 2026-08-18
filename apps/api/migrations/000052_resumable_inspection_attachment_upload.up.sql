ALTER TABLE upload_sessions
    ADD COLUMN IF NOT EXISTS session_epoch bigint NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS part_size_bytes bigint,
    ADD COLUMN IF NOT EXISTS whole_file_sha256 text,
    ADD COLUMN IF NOT EXISTS object_version_id text,
    ADD COLUMN IF NOT EXISTS object_etag text;

ALTER TABLE upload_sessions
    DROP CONSTRAINT IF EXISTS upload_sessions_upload_state_check;

ALTER TABLE upload_sessions
    ADD CONSTRAINT upload_sessions_upload_state_check
    CHECK (upload_state IN ('PENDING', 'OPEN', 'UPLOADING', 'PARTIALLY_COMMITTED', 'COMPLETING', 'COMPLETED', 'EXPIRED', 'ABORTED', 'QUARANTINED'));

CREATE TABLE IF NOT EXISTS inspection_attachment_upload_parts (
    id text PRIMARY KEY,
    upload_session_id text NOT NULL REFERENCES upload_sessions(id),
    session_epoch bigint NOT NULL CHECK (session_epoch > 0),
    part_number bigint NOT NULL CHECK (part_number > 0),
    byte_size bigint NOT NULL CHECK (byte_size >= 0),
    part_sha256 text NOT NULL,
    part_object_key text NOT NULL,
    part_state text NOT NULL CHECK (part_state IN ('OPEN', 'ACKNOWLEDGED', 'QUARANTINED')),
    object_version_id text,
    object_etag text,
    created_at timestamptz NOT NULL,
    acknowledged_at timestamptz,
    UNIQUE (upload_session_id, session_epoch, part_number),
    UNIQUE (upload_session_id, session_epoch, part_number, part_sha256)
);

CREATE INDEX IF NOT EXISTS inspection_attachment_upload_parts_session_idx
    ON inspection_attachment_upload_parts (upload_session_id, session_epoch, part_number);
