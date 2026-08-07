-- Exercise review commands are append-only, but replay identity must be
-- durable independently from the event identifier.  Backfill legacy rows
-- before enforcing the unique operation/idempotency boundaries.
ALTER TABLE canonical_exercise_question_review_events
    ADD COLUMN IF NOT EXISTS operation_id text,
    ADD COLUMN IF NOT EXISTS idempotency_key text;

ALTER TABLE canonical_exercise_question_review_events
    DISABLE TRIGGER canonical_exercise_question_review_events_append_only;

UPDATE canonical_exercise_question_review_events
SET operation_id = COALESCE(NULLIF(operation_id, ''), event_id),
    idempotency_key = COALESCE(NULLIF(idempotency_key, ''), 'legacy:' || event_id)
WHERE operation_id IS NULL OR operation_id = '' OR idempotency_key IS NULL OR idempotency_key = '';

ALTER TABLE canonical_exercise_question_review_events
    ENABLE TRIGGER canonical_exercise_question_review_events_append_only;

ALTER TABLE canonical_exercise_question_review_events
    ALTER COLUMN operation_id SET NOT NULL,
    ALTER COLUMN idempotency_key SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS canonical_exercise_question_review_events_operation_idx
    ON canonical_exercise_question_review_events (operation_id);
CREATE UNIQUE INDEX IF NOT EXISTS canonical_exercise_question_review_events_idempotency_idx
    ON canonical_exercise_question_review_events (idempotency_key);
