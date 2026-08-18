ALTER TABLE idempotency_responses
    ADD COLUMN IF NOT EXISTS terminal_at timestamptz;

UPDATE idempotency_responses
SET terminal_at = created_at + INTERVAL '400 days'
WHERE terminal_at IS NULL;

ALTER TABLE idempotency_responses
    ALTER COLUMN terminal_at SET NOT NULL;

ALTER TABLE idempotency_responses
    ALTER COLUMN terminal_at SET DEFAULT (now() + INTERVAL '400 days');

DO $idempotency_retention$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'idempotency_responses'::regclass
          AND conname = 'idempotency_responses_terminal_after_created'
    ) THEN
        ALTER TABLE idempotency_responses
            ADD CONSTRAINT idempotency_responses_terminal_after_created
            CHECK (terminal_at >= created_at);
    END IF;
END
$idempotency_retention$;

CREATE INDEX IF NOT EXISTS idempotency_responses_terminal_at_idx
    ON idempotency_responses (terminal_at);

ALTER TABLE sync_cursor_tokens
    ADD COLUMN IF NOT EXISTS profile_key_id text NOT NULL DEFAULT '';

ALTER TABLE sync_cursors
    ADD COLUMN IF NOT EXISTS profile_key_id text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS sync_cursor_tokens_profile_scope_idx
    ON sync_cursor_tokens (subject_id, organization_id, package_id, grant_id, device_id, profile_key_id);

CREATE INDEX IF NOT EXISTS sync_cursors_profile_scope_idx
    ON sync_cursors (subject_id, device_id, profile_key_id);
