import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

test('Task 6 reconciliation wrapper accepts only two readable manifest paths', async () => {
  const script = await readFile(new URL('../scripts/reconcile-aviacore-feed.sh', import.meta.url), 'utf8');
  assert.match(script, /usage: \$0 <producer-manifest\.json> <aviacore-manifest\.json>/);
  assert.match(script, /\[\[ \$# -ne 2 \]\]/);
  assert.match(script, /data-feed-reconcile/);
  assert.match(script, /PRODUCER_MANIFEST=/);
  assert.match(script, /AVIACORE_MANIFEST=/);
  assert.doesNotMatch(script, /curl|wget|docker compose/);
});
