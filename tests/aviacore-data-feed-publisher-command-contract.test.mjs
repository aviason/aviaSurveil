import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const script = await readFile(new URL('../scripts/test-aviacore-data-feed-publisher.sh', import.meta.url), 'utf8');

test('Task 5 publisher acceptance command owns a fresh evidence root and cleanup', () => {
  for (const required of [
    'mktemp -d /private/tmp/aviasurveil360-data-feed-task5.',
    'AVIA_DATA_FEED_EVIDENCE_ROOT must be unset',
    'go -C "${REPOSITORY_ROOT}/apps/api" test -race -count=1 ./internal/datafeed ./cmd/data-feed-worker',
    'docker compose --project-name "${COMPOSE_PROJECT}"',
    'TestDataFeedPublisherConcreteLeaseAndReceiptPersistence',
    'scripts/check-sqlc.sh',
    'rm -rf "${EVIDENCE_ROOT}"',
    'candidate-only',
    'production-ready: not established',
  ]) assert.ok(script.includes(required), `missing ${required}`);
});
