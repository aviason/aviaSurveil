ALTER TABLE document_render_jobs
    ADD COLUMN IF NOT EXISTS lease_owner text,
    ADD COLUMN IF NOT EXISTS lease_generation bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS lease_expires_at timestamptz;

ALTER TABLE outbox_messages
    ADD COLUMN IF NOT EXISTS lease_generation bigint NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS document_render_jobs_lease_claimable_idx
    ON document_render_jobs (status, lease_expires_at, created_at, id)
    WHERE status IN ('PENDING', 'FAILED', 'RUNNING');

CREATE TABLE IF NOT EXISTS document_render_job_dispositions (
    id text PRIMARY KEY,
    job_id text NOT NULL REFERENCES document_render_jobs(id),
    disposition text NOT NULL CHECK (disposition IN (
        'SUPERSEDED_GOTENBERG', 'DEAD_LETTER', 'MANUAL_RETRY'
    )),
    attempt_count integer NOT NULL CHECK (attempt_count >= 0),
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (job_id, disposition, attempt_count)
);

CREATE INDEX IF NOT EXISTS document_render_job_dispositions_job_idx
    ON document_render_job_dispositions (job_id, created_at, id);

DROP TRIGGER IF EXISTS document_render_job_dispositions_append_only
    ON document_render_job_dispositions;
CREATE TRIGGER document_render_job_dispositions_append_only
BEFORE UPDATE OR DELETE ON document_render_job_dispositions
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

-- Existing unfinished jobs were bound to the retired Gotenberg renderer. Keep
-- successful artifacts and every legacy job/source/outbox row untouched. A
-- pending/failed job, or a running job whose known lease has expired, receives
-- only an append-only disposition. A running job without an expired lease is
-- deliberately left for an explicit drain; the native claim query excludes
-- every legacy snapshot until that drain is complete.
INSERT INTO document_render_job_dispositions
    (id, job_id, disposition, attempt_count, details, created_at)
SELECT
    job.id || '-superseded-gotenberg',
    job.id,
    'SUPERSEDED_GOTENBERG',
    job.attempt_count,
    jsonb_build_object('reason', 'legacy-renderer-retired', 'status', job.status),
    now()
FROM document_render_jobs job
WHERE job.status <> 'SUCCEEDED'
  AND NOT (job.input_snapshot ? 'source')
  AND (
      job.status IN ('PENDING', 'FAILED')
      OR (job.status = 'RUNNING' AND job.lease_expires_at IS NOT NULL
          AND job.lease_expires_at <= now())
  )
ON CONFLICT (job_id, disposition, attempt_count) DO NOTHING;
