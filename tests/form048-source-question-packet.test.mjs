import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import test from 'node:test';

const repositoryRoot = process.cwd();
const packageRoot = join(repositoryRoot, 'deliverables/form-048-admin-review-request-2026-08-01');
const packet = JSON.parse(readFileSync(join(packageRoot, 'FORM_048_28_SOURCE_QUESTIONS.json'), 'utf8'));
const template = JSON.parse(readFileSync(join(packageRoot, 'FORM_048_ADMIN_DECISION_TEMPLATE.json'), 'utf8'));

const expectedCodes = [
  '100', '102', '104', '106', '202', '204', '206', '208', '210', '302', '304', '306',
  '402', '502', '504', '506', '508', '602', '604', '802', '804', '806', '1002', '1004',
  '1102', '1104', '1106', '1108',
];

test('source-backed Form 048 packet contains all 28 literal questions and remains fail-closed', () => {
  assert.equal(packet.packageVersion, 'AGA_EXTRACTION_REVIEW_V1');
  assert.equal(packet.status, 'PENDING_ADMIN_REVIEW');
  assert.equal(packet.bindingState, 'PENDING_CURRENT_SERVER_PACKET_BINDING');
  assert.equal(packet.source.questionCount, 28);
  assert.equal(packet.questions.length, 28);
  assert.deepEqual(packet.questions.map((question) => question.formCode.replace('AGA 048.', '')), expectedCodes);
  assert.equal(new Set(packet.questions.map((question) => question.proposalId)).size, 28);
  for (const question of packet.questions) {
    assert.match(question.originalText, /\?$/);
    assert.ok(question.originalText.length > 20);
    assert.ok(question.page >= 1 && question.page <= 9);
    assert.ok(question.sourceLocator.includes(question.formCode));
    assert.ok(question.sourceRefs.length >= 2);
    assert.equal(question.decisionKind, '');
    assert.equal(question.resultHistoryState, 'NOT_SUPPLIED');
    assert.equal(question.sourceMappingState, 'SOURCE_MAPPING_REQUIRED');
  }
  assert.equal(template.questionBoundaryDecisionSet.decisions.length, 28);
  assert.ok(template.questionBoundaryDecisionSet.decisions.every((decision) => decision.decisionKind === '' && decision.resultHistoryState === 'NOT_SUPPLIED'));
  assert.equal(template.candidateAuthorization.authorize, false);
  assert.equal(template.candidateAuthorization.sourceMappingAuthorized, false);
  assert.equal(template.candidateAuthorization.publicationAuthorized, false);
});

test('source-backed PDF renders as a non-empty multi-page review artifact', () => {
  const pdfPath = join(packageRoot, 'FORM_048_28_SORU_ADMIN_KARAR_FORMU_TR.pdf');
  assert.ok(statSync(pdfPath).size > 100_000);
  const info = execFileSync('pdfinfo', [pdfPath], { encoding: 'utf8' });
  assert.match(info, /Pages:\s+11/);
  assert.match(info, /Page size:\s+594\.96 x 841\.92 pts \(A4\)/);
});
