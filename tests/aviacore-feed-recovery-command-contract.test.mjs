import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

test('Task 6 recovery acceptance command requires two manifests and owns fresh cleanup', async () => {
  const script = await readFile(new URL('../scripts/test-aviacore-feed-recovery.sh', import.meta.url), 'utf8');
  for (const required of [
    'AVIA_RECOVERY_PRODUCER_MANIFEST',
    'AVIA_RECOVERY_AVIACORE_MANIFEST',
    'mktemp -d /private/tmp/aviasurveil360-datafeed-task6',
    'PRODUCER_MANIFEST="$(cd "$(dirname "${AVIA_RECOVERY_PRODUCER_MANIFEST}")" && pwd)/$(basename "${AVIA_RECOVERY_PRODUCER_MANIFEST}")"',
    'AVIACORE_MANIFEST="$(cd "$(dirname "${AVIA_RECOVERY_AVIACORE_MANIFEST}")" && pwd)/$(basename "${AVIA_RECOVERY_AVIACORE_MANIFEST}")"',
    'data-feed-reconcile',
    'datafeed_replay',
    'rm -rf "${EVIDENCE_ROOT}"',
    'candidate-only',
    'production-ready: not established',
  ]) {
    assert.match(script, new RegExp(required.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }
  assert.match(script, /AVIA_RECOVERY_PRODUCER_MANIFEST must name a readable manifest/);
  assert.match(script, /AVIA_RECOVERY_AVIACORE_MANIFEST must name a readable manifest/);
});
