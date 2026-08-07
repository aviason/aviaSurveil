import test from 'node:test';
import assert from 'node:assert/strict';
import { assembleOpenApi } from '../../../scripts/bundle-openapi.mjs';

const source = assembleOpenApi();

test('canonical AGA successor exposes catalog, scope, and review boundaries', () => {
  assert.ok(source.paths['/v1/question-catalogs/{catalogVersion}/questions'], 'catalog list route');
  assert.ok(source.paths['/v1/audit-scopes/{scopeId}/preview'], 'scope preview route');
  assert.ok(source.paths['/v1/department-manager/question-review/exercise-commands'], 'exercise question review command route');
  assert.ok(source.paths['/v1/department-manager/question-review/governed-commands'], 'governed question review command route');
  assert.ok(source.paths['/v1/audit-assignments/{assignmentId}/preparation-confirmations'], 'preparation confirmation route');
  assert.ok(source.paths['/v1/audit-assignments/{assignmentId}/materializations'], 'canonical materialization route');
  assert.ok(source.paths['/v1/audit-assignments/{assignmentId}/question-coverage'], 'question coverage route');
  assert.ok(source.paths['/v1/audit-scope-options'], 'server-owned New Audit selector route');
  assert.ok(source.components.schemas.CanonicalQuestionCatalogEntry, 'catalog entry schema');
  assert.ok(source.components.schemas.ExerciseQuestionReviewCommandInput, 'exercise-only review command schema');
  assert.ok(source.components.schemas.GovernedQuestionReviewCommandInput, 'governed review command schema');
  assert.ok(source.components.schemas.AuditScopeSelectionDigest, 'selection digest schema');
  assert.ok(source.components.schemas.QuestionReviewMode, 'review mode schema');
  assert.ok(source.components.schemas.ConfirmAuditPreparationInput, 'preparation confirmation input');
  assert.ok(source.components.schemas.MaterializeCanonicalAuditInput, 'canonical materialization input');
});

test('canonical contract names the preprod exercise boundary', () => {
  const usage = source.components.schemas.QuestionUsageClass;
  assert.deepEqual(usage.enum, ['GOVERNED_OPERATIONAL', 'PREPROD_EXERCISE']);
  assert.equal(source.info['x-preprod-exercise-publication'], 'forbidden');
});
