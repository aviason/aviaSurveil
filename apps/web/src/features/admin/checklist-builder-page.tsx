import { useState } from "react";

import type { AdminProposedInspectionQuestionView, AdminRegulatoryMappingView, AdminTemplateVersionView } from "../../backend/backend";
import { AdminError, AdminPage, DisabledAdminAction, useAdminLoad, useAdminWorkspace } from "./admin-workspace-shared";

interface RegulatoryQuestionTrace {
  mapping: AdminRegulatoryMappingView;
  proposal: AdminProposedInspectionQuestionView;
}

export function ChecklistBuilderPage() {
  const backend = useAdminWorkspace();
  const [selectedQuestionId, setSelectedQuestionId] = useState("");
  const [commandError, setCommandError] = useState<string | null>(null);
  const templateLoad = useAdminLoad(() => backend.getTemplate({ templateId: "TPL-CABIN-2026" }), [backend]);
  const questionLoad = useAdminLoad(() => backend.listQuestions({}), [backend]);
  const regulatoryLoad = useAdminLoad(() => backend.listRegulatoryReferences({ status: "ACTIVE" }), [backend]);
  const template = templateLoad.data;
  const draft = template?.versions.find((version) => version.status === "DRAFT") ?? null;
  const published = template?.versions.find((version) => version.id === "CTV-CABIN-1") ?? null;
  const questions = new Map(questionLoad.data?.items.map((question) => [question.id, question]) ?? []);
  const traceByQuestionId = new Map<string, RegulatoryQuestionTrace>();
  for (const reference of regulatoryLoad.data?.items ?? []) {
    for (const mapping of reference.mappings) {
      for (const proposal of mapping.proposedQuestions) traceByQuestionId.set(proposal.id, { mapping, proposal });
    }
  }
  const publishedTrace = published?.questionIds.map((questionId) => traceByQuestionId.get(questionId)).filter((trace): trace is RegulatoryQuestionTrace => Boolean(trace)) ?? [];
  const pilotMapping = publishedTrace.at(0)?.mapping ?? null;
  const scopeRecommendation = pilotMapping?.scopeRecommendation ?? null;
  const scopeByQuestionId = new Map(
    scopeRecommendation?.questionRecommendations.map((recommendation) => [
      recommendation.questionId,
      recommendation,
    ]) ?? [],
  );
  const available = questionLoad.data?.items.filter((question) => !draft?.questionIds.includes(question.id)) ?? [];

  async function run(command: () => Promise<AdminTemplateVersionView>) {
    setCommandError(null);
    try { await command(); templateLoad.reload(); questionLoad.reload(); regulatoryLoad.reload(); } catch (cause) { setCommandError(cause instanceof Error ? cause.message : "Draft update failed."); }
  }

  async function addQuestion(version: AdminTemplateVersionView, questionId: string) {
    setCommandError(null);
    try {
      await backend.addDraftQuestion({ templateId: version.templateId, draftVersionId: version.id, questionId, expectedRevision: version.revision, idempotencyKey: `ADMIN-${version.id}-ADD-${questionId}` });
      setSelectedQuestionId("");
      templateLoad.reload();
      questionLoad.reload();
    } catch (cause) {
      setCommandError(cause instanceof Error ? cause.message : "Draft update failed.");
    }
  }

  return (
    <AdminPage testId="admin-checklist-builder-page" routeLabel="Checklist Builder" title="Checklist Builder" description="Configure one exact working Draft without changing the immutable published version.">
      <AdminError message={templateLoad.error ?? questionLoad.error ?? commandError} />
      <section className="admin-template-identity" aria-label="Template identity">
        <div><span>Template master</span><b>TPL-CABIN-2026</b></div><div><span>Immutable published version</span><b>CTV-CABIN-1</b></div><div><span>Published owner</span><b>Department Manager</b></div>
      </section>
      <AdminError message={regulatoryLoad.error} />
      {pilotMapping ? (
        <section className="admin-regulatory-pilot" aria-labelledby="regulatory-pilot-title">
          <header>
            <div><p className="eyebrow">Candidate-only regulatory trace</p><h2 id="regulatory-pilot-title">OPS · Air Operator (AOC) cabin/ramp pilot</h2><small>{pilotMapping.id}</small></div>
            <span className="admin-review-badge admin-review-badge--expert_review_required">{pilotMapping.reviewStatus.replaceAll("_", " ")}</span>
          </header>
          <dl>
            <div><dt>ICAO starting point</dt><dd>PQ {pilotMapping.protocolQuestionId} · {pilotMapping.criticalElement}</dd></div>
            <div><dt>Checklist coverage</dt><dd>{publishedTrace.length} of {published?.questionIds.length ?? 0} published questions traced</dd></div>
            <div><dt>Service provider</dt><dd>{pilotMapping.serviceProviderTypes.join(", ")}</dd></div>
            <div><dt>Applicable regulation scope</dt><dd>{pilotMapping.applicableRegulations.join(" · ")}</dd></div>
          </dl>
          {pilotMapping.sourceGap ? <aside className="admin-source-gap"><b>Expert validation gate</b><p>{pilotMapping.sourceGap}</p></aside> : null}
          <p>{pilotMapping.whyIncluded}</p>
        </section>
      ) : null}
      {scopeRecommendation ? (
        <section className="admin-scope-recommendation" aria-labelledby="scope-recommendation-title">
          <header>
            <div>
              <p className="eyebrow">Inspection-efficiency recommendation</p>
              <h2 id="scope-recommendation-title">Recommended scope for the current scenario</h2>
              <small>{scopeRecommendation.id} · {scopeRecommendation.status.replaceAll("_", " ")}</small>
            </div>
            <span className="admin-review-badge admin-review-badge--expert_review_required">{scopeRecommendation.historyState.replaceAll("_", " ")}</span>
          </header>
          <p>The recommendation concentrates effort without silently removing controls. The Inspector or Department Manager must approve and record any scope adjustment.</p>
          <div className="admin-regulatory-detail-grid">
            <section><h3>Signals used</h3><ul>{scopeRecommendation.signals.map((signal) => <li key={signal}>{signal}</li>)}</ul></section>
            <section><h3>Hard guardrails</h3><ul>{scopeRecommendation.guardrails.map((guardrail) => <li key={guardrail}>{guardrail}</li>)}</ul></section>
          </div>
        </section>
      ) : null}
      {!draft && template ? <button onClick={() => void run(() => backend.createDraft({ templateId: template.id, expectedRevision: template.revision, idempotencyKey: "ADMIN-TPL-CABIN-2026-DRAFT-2", changeReason: "Create a browser-local working checklist Draft." }))} type="button">Create working Draft</button> : null}
      {published ? (
        <section className="admin-record-card">
          <h2>Published / locked</h2>
          <p>{published.id} · {published.questionIds.length} exact questions · Revision {published.revision}</p>
          {publishedTrace.length > 0 ? (
            <ol className="admin-published-trace-list">
              {published.questionIds.map((questionId) => {
                const question = questions.get(questionId);
                const trace = traceByQuestionId.get(questionId);
                const scope = scopeByQuestionId.get(questionId);
                return (
                  <li key={questionId}>
                    <b>{question?.prompt ?? trace?.proposal.prompt ?? "Question unavailable"}</b>
                    <small>{questionId}</small>
                    {scope ? (
                      <div className="admin-question-scope">
                        <span>{scope.classification.replaceAll("_", " ")}</span>
                        <p>{scope.rationale}</p>
                        <small><b>History basis:</b> {scope.historyBasis}</small>
                      </div>
                    ) : null}
                    {trace ? (
                      <details className="admin-question-trace">
                        <summary>Regulatory trace · PQ {trace.mapping.protocolQuestionId} · {trace.mapping.criticalElement}</summary>
                        <p><b>Verification method:</b> {trace.proposal.verificationMethod}</p>
                        <p><b>Expected evidence:</b> {trace.proposal.evidenceExamples.join(" · ")}</p>
                        <p><b>Why included:</b> {trace.proposal.whyIncluded}</p>
                      </details>
                    ) : <p>Regulatory trace has not been configured for this question.</p>}
                  </li>
                );
              })}
            </ol>
          ) : null}
          <DisabledAdminAction label={`Edit ${published.id}`} reason={`${published.id} is PUBLISHED and its question array is immutable.`} />
        </section>
      ) : null}
      {draft ? (
        <section className="admin-builder-draft" aria-label={`Working Draft ${draft.id}`}>
          <header><div><h2>Working Draft</h2><p>{draft.id} · Revision {draft.revision} · Owner {draft.owner}</p></div><DisabledAdminAction label={`Publish ${draft.id}`} reason={`${draft.id} is DRAFT and Department Manager owns publishing after approval; Admin Preview cannot publish or submit it.`} /></header>
          <div className="admin-builder-add"><label>Question to add<select aria-label="Question to add" onChange={(event) => setSelectedQuestionId(event.target.value)} value={selectedQuestionId}><option value="">Select an exact question</option>{available.map((question) => <option key={question.id} value={question.id}>{question.id} — {question.prompt}</option>)}</select></label><button disabled={!selectedQuestionId || !available.some((question) => question.id === selectedQuestionId)} aria-label={!selectedQuestionId || !available.some((question) => question.id === selectedQuestionId) ? `Add question to ${draft.id} unavailable: select an exact Question Bank record.` : `Add ${selectedQuestionId} to ${draft.id}`} title={!selectedQuestionId || !available.some((question) => question.id === selectedQuestionId) ? `Select an exact Question Bank record for ${draft.id}.` : undefined} onClick={() => selectedQuestionId && void addQuestion(draft, selectedQuestionId)} type="button">Add question</button></div>
          <ol className="admin-builder-list">
            {draft.questionIds.map((questionId, index) => {
              const question = questions.get(questionId);
              const trace = traceByQuestionId.get(questionId);
              const scope = scopeByQuestionId.get(questionId);
              const upReason = `${questionId} is already first in ${draft.id}.`;
              const downReason = `${questionId} is already last in ${draft.id}.`;
              return <li key={questionId}><div><span className="admin-order">{index + 1}</span><div className="admin-builder-question-copy"><p><b>{question?.prompt ?? "Question unavailable"}</b><small>{questionId}</small><span>{question?.configuredReference}</span></p>{scope ? <div className="admin-question-scope"><span>{scope.classification.replaceAll("_", " ")}</span><p>{scope.rationale}</p></div> : null}{trace ? <details className="admin-question-trace"><summary>Regulatory trace · PQ {trace.mapping.protocolQuestionId} · {trace.mapping.criticalElement}</summary><p><b>Verify:</b> {trace.proposal.verificationMethod}</p><p><b>Evidence:</b> {trace.proposal.evidenceExamples.join(" · ")}</p></details> : null}</div></div><div className="admin-builder-row-actions"><button aria-label={index === 0 ? `Move ${questionId} up unavailable: ${upReason}` : `Move ${questionId} up in ${draft.id}`} disabled={index === 0} title={index === 0 ? upReason : undefined} onClick={() => void run(() => backend.moveDraftQuestion({ templateId: draft.templateId, draftVersionId: draft.id, questionId, direction: "UP", expectedRevision: draft.revision, idempotencyKey: `ADMIN-${draft.id}-MOVE-${questionId}-UP-R${draft.revision}` }))} type="button">↑ <span>Up</span></button><button aria-label={index === draft.questionIds.length - 1 ? `Move ${questionId} down unavailable: ${downReason}` : `Move ${questionId} down in ${draft.id}`} disabled={index === draft.questionIds.length - 1} title={index === draft.questionIds.length - 1 ? downReason : undefined} onClick={() => void run(() => backend.moveDraftQuestion({ templateId: draft.templateId, draftVersionId: draft.id, questionId, direction: "DOWN", expectedRevision: draft.revision, idempotencyKey: `ADMIN-${draft.id}-MOVE-${questionId}-DOWN-R${draft.revision}` }))} type="button">↓ <span>Down</span></button></div></li>;
            })}
          </ol>
        </section>
      ) : null}
    </AdminPage>
  );
}
