import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const artifactRoot = new URL('../deliverables/aga-question-classification-candidate-2026-08-03/', import.meta.url);
const artifactNames = [
  'aggregates.json',
  'ambiguity-review.csv',
  'batch-manifest.json',
  'manifest.json',
  'pass-isolation-cleanup.json',
  'pass-one-results.json',
  'pass-one-run.json',
  'pass-two-results.json',
  'pass-two-run.json',
  'question-classifications.csv',
  'question-classifications.json',
  'reconciliation.json',
];

test('Task 2 reconciliation artifact is sealed, bounded, and text-free', async () => {
  const rawArtifacts = new Map(
    await Promise.all(
      artifactNames.map(async (name) => [name, await readFile(new URL(name, artifactRoot), 'utf8')]),
    ),
  );
  for (const raw of rawArtifacts.values()) {
    assert.equal(raw.includes('questionBody'), false);
    assert.equal(raw.includes('/private/'), false);
  }

  const cleanup = JSON.parse(rawArtifacts.get('pass-isolation-cleanup.json'));
  assert.equal(cleanup.status, 'SEALED');
  assert.equal(cleanup.result, 'AGA_PASS_VALIDATED');
  assert.equal(cleanup.candidate.semanticEntries, 27);
  assert.equal(cleanup.candidate.transportNoise, 28);
  assert.equal(cleanup.candidate.batchCount, 25);
  assert.equal(cleanup.candidate.recordCount, 1310);
  assert.equal(cleanup.challenge.batchCount, 25);
  assert.equal(cleanup.challenge.recordCount, 1310);
  assert.equal(cleanup.challenge.semanticEntries, 27);
  assert.equal(cleanup.challenge.transportNoise, 28);
  assert.equal(cleanup.candidate.privateRootRemoved, true);
  assert.equal(cleanup.challenge.privateRootRemoved, true);
  assert.deepEqual(cleanup.cleanup, {
    privateRootRemoved: true,
    filesRemaining: 0,
    directoriesRemaining: 0,
    processesRemaining: 0,
  });

  const manifest = JSON.parse(rawArtifacts.get('manifest.json'));
  assert.equal(manifest.status, 'SEALED');
  assert.equal(manifest.candidateOnly, true);
  assert.equal(manifest.batchCount, 25);
  assert.equal(manifest.itemCount, 1310);
  assert.equal(manifest.passProposalRecordCount, 2620);
  assert.equal(
    manifest.promptDigest,
    'sha256:a3e1a02c64d403bbca346b6bb33f4dcd509c06fafcc082d197120098322dd1b1',
  );
  assert.equal(
    manifest.passOneSealDigest,
    'sha256:826ab71f897576ccf1cbd7cb96cc2bac58365e19434713c07fb86d172482d279',
  );
  assert.equal(
    manifest.passTwoSealDigest,
    'sha256:7af40f0e08b1fd5514a6a982f43407fb65056aa2e114da227fe35f5e7ef90c21',
  );
  assert.equal(manifest.candidateSource.sha256, cleanup.candidate.sourceArchiveSha256);
  assert.equal(manifest.challengeSource.sha256, cleanup.challenge.sourceArchiveSha256);
  assert.equal(manifest.files.length, 11);

  const passOneRun = JSON.parse(rawArtifacts.get('pass-one-run.json'));
  const passTwoRun = JSON.parse(rawArtifacts.get('pass-two-run.json'));
  assert.equal(
    passOneRun.modelDescriptorDigest,
    'sha256:89c5ce043615b79d0dd5cba75b9ab3953fd636d8b9ad17e468452f4dfbf3d47a',
  );
  assert.equal(
    passTwoRun.modelDescriptorDigest,
    'sha256:0572227e217aa0e2a4c5c123d9a36666689d2fe5200bfccdfbb0cc5756d8d9e6',
  );
  assert.equal(passOneRun.passSealReceipt.promptDigest, passTwoRun.passSealReceipt.promptDigest);

  const reconciliation = JSON.parse(rawArtifacts.get('reconciliation.json'));
  assert.equal(reconciliation.state, 'SEALED');
  assert.equal(reconciliation.aggregate.itemCount, 1310);
  assert.equal(reconciliation.aggregate.passProposalRecordCount, 2620);
  assert.equal(reconciliation.classificationRunId, manifest.classificationRunId);
});
