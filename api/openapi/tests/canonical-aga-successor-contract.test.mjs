import test from 'node:test';
import assert from 'node:assert/strict';
import { assembleOpenApi } from '../../../scripts/bundle-openapi.mjs';

const source = assembleOpenApi();

test('canonical AGA successor exposes catalog, scope, and review boundaries', () => {
  assert.ok(source.paths['/v1/question-catalogs/{catalogVersion}/questions'], 'catalog list route');
  assert.ok(source.paths['/v1/audit-scopes/{scopeId}/preview'], 'scope preview route');
  assert.ok(source.paths['/v1/department-manager/question-review/commands'], 'question review command route');
  assert.ok(source.components.schemas.CanonicalQuestionCatalogEntry, 'catalog entry schema');
  assert.ok(source.components.schemas.AuditScopeSelectionDigest, 'selection digest schema');
  assert.ok(source.components.schemas.QuestionReviewMode, 'review mode schema');
});

test('canonical contract names the preprod exercise boundary', () => {
  const usage = source.components.schemas.QuestionUsageClass;
  assert.deepEqual(usage.enum, ['GOVERNED_OPERATIONAL', 'PREPROD_EXERCISE']);
  assert.equal(source.info['x-preprod-exercise-publication'], 'forbidden');
});
