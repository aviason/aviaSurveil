import { useEffect, useRef, useState } from "react";

import { useBackendForRole } from "../../app/providers";
import type { AdminProposedInspectionQuestionView, AdminRegulatoryMappingView, AdminTemplateVersionView, ChecklistImportBatchView, GovernedCandidateBundleInput, GovernedGenerationRunView, GovernedQuestionView, GovernedValidationIssue } from "../../backend/backend";
import { GovernedValidationError } from "../../backend/backend-contracts";
import {
  SYNTHETIC_EDITED_RATIONALE,
  SYNTHETIC_GOVERNED_BUNDLE,
  SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
  SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE,
} from "../../backend/governed-synthetic-profile";
import { AdminError, AdminPage, DisabledAdminAction, useAdminLoad, useAdminWorkspace } from "./admin-workspace-shared";
import { ChecklistIntakePanel } from "./checklist-intake-panel";
import { AGACandidateDemoPanel } from "./aga-candidate-demo-panel";

interface RegulatoryQuestionTrace {
  mapping: AdminRegulatoryMappingView;
  proposal: AdminProposedInspectionQuestionView;
}

export function ChecklistBuilderRoute() {
  const adminBackend = useBackendForRole("admin");
  const capability = adminBackend.agaCandidateDemo;
  const [demoState, setDemoState] = useState<"checking" | "available" | "unavailable">(
    capability ? "checking" : "unavailable",
  );

  useEffect(() => {
    if (!capability) return;
    const controller = new AbortController();
    void capability.capability({}, { signal: controller.signal })
      .then((result) => setDemoState(result.available ? "available" : "unavailable"))
      .catch(() => {
        if (!controller.signal.aborted) setDemoState("unavailable");
      });
    return () => controller.abort();
  }, [capability]);

  if (demoState === "available") {
    return (
      <AdminPage
        testId="admin-checklist-builder-page"
        routeLabel="Checklist Builder"
        title="Checklist Builder"
        description="Review the immutable local-preprod AGA candidate projection."
      >
        <AGACandidateDemoPanel capability={capability} />
      </AdminPage>
    );
  }
  if (demoState === "checking") {
    return (
      <AdminPage
        testId="admin-checklist-builder-page"
        routeLabel="Checklist Builder"
        title="Checklist Builder"
        description="Checking the sealed local-preprod capability."
      >
        <p role="status">Checking candidate demo availability…</p>
      </AdminPage>
    );
  }
  return <ChecklistBuilderPage />;
}

function readableGovernedState(value: string | undefined) {
  return value?.replaceAll("_", " ") ?? "Not recorded";
}

function GovernedQuestionGovernance({ question }: { question: GovernedQuestionView }) {
  const scope = question.scopeRecommendation;
  const trace = question.regulatoryTrace;
  const gap = trace.state === "SOURCE_MAPPING_REQUIRED";
  const guardrails = [
    scope.guardrails.mandatoryControl ? "Mandatory control" : null,
    scope.guardrails.safetyCritical ? "Safety-critical" : null,
    scope.guardrails.unknownHistory ? "Unknown history" : null,
    scope.guardrails.sourceChanged ? "Source changed" : null,
    scope.guardrails.overdueControl ? "Overdue control" : null,
    scope.guardrails.automaticDeferralPermitted ? "Automatic deferral permitted" : "Automatic deferral blocked",
  ].filter((value): value is string => Boolean(value));

  return (
    <details className="admin-question-trace" open>
      <summary>{String(question.questionId)} · {String(question.prompt)}</summary>
      <dl className="admin-regulatory-detail-grid">
        <div><dt>Question origin</dt><dd>{readableGovernedState(question.origin)}</dd></div>
        <div><dt>Scope classification</dt><dd>{readableGovernedState(scope.classification)}</dd></div>
        <div><dt>Operational-history basis</dt><dd>{scope.operationalHistoryBasis}</dd></div>
        <div><dt>Scope review state</dt><dd>{readableGovernedState(scope.approvalReviewState)}</dd></div>
      </dl>
      <p><b>Inclusion / deferral rationale:</b> {scope.rationale}</p>
      <p><b>Input signals:</b> {scope.inputSignals.join(" · ")}</p>
      <p><b>Guardrails:</b> {guardrails.join(" · ")}</p>
      <p><b>Automatic deferral:</b> {scope.automaticDeferral ? "Requested" : "Not requested"}</p>
      {gap ? (
        <aside className="admin-source-gap">
          <b>SOURCE_MAPPING_REQUIRED</b>
          <p>This Draft remains repairable, but it cannot be validated, automatically deferred, included in an executable published Audit package, or published.</p>
        </aside>
      ) : (
        <>
          <dl className="admin-regulatory-detail-grid">
            <div><dt>Source title</dt><dd>{trace.sourceTitle}</dd></div>
            <div><dt>Source version / hash</dt><dd>{trace.immutableVersion} · {trace.sha256}</dd></div>
            <div><dt>Page / locator / clause</dt><dd>{trace.page} · {trace.locator} · {trace.clause}</dd></div>
            <div><dt>Source type / section</dt><dd>{trace.sourceType} · {trace.section}</dd></div>
            <div><dt>Applicability</dt><dd>{trace.applicability}</dd></div>
            <div><dt>Currentness / technical review</dt><dd>{readableGovernedState(trace.currentnessState)} · {readableGovernedState(trace.technicalReviewState)}</dd></div>
          </dl>
          <p><b>National reference:</b> {trace.nationalReference}</p>
          <p><b>Controlled CAA procedure mapping:</b> {trace.controlledCaaProcedureMapping}</p>
          <p><b>Verification objective:</b> {trace.verificationObjective}</p>
          <p><b>Expected Evidence:</b> {trace.expectedEvidence?.join(" · ")}</p>
          <p><b>Question citations:</b> {JSON.stringify(question.citations)}</p>
        </>
      )}
      {question.reconciliation ? (
        <section className="admin-question-reconciliation" aria-label={`Hybrid reconciliation ${question.questionId}`}>
          <h3>Legacy candidate comparison</h3>
          <p><b>Legacy question / wording:</b> {question.reconciliation.legacyQuestionId} · {question.reconciliation.legacyWording}</p>
          <p><b>Legacy operational intent / result history:</b> {question.reconciliation.legacyOperationalIntent} · {question.reconciliation.legacyResultHistory}</p>
          <p><b>Wording / Evidence / applicability / scope changed:</b> {question.reconciliation.wordingChanged ? "yes" : "no"} · {question.reconciliation.evidenceChanged ? "yes" : "no"} · {question.reconciliation.applicabilityChanged ? "yes" : "no"} · {question.reconciliation.scopeChanged ? "yes" : "no"}</p>
          <p><b>Current trace controls:</b> {question.reconciliation.currentWording} · {question.reconciliation.currentExpectedEvidence.join(" · ")} · {question.reconciliation.currentApplicability} · {readableGovernedState(question.reconciliation.currentScopeClassification)}</p>
        </section>
      ) : null}
    </details>
  );
}

export function ChecklistBuilderPage() {
  const backend = useAdminWorkspace();
  const adminBackend = useBackendForRole("admin");
  const [selectedQuestionId, setSelectedQuestionId] = useState("");
  // A governed candidate is an explicit imported/retrieved immutable
  // artifact. Do not manufacture a default ID: on a clean HTTP profile that
  // would issue a 404 before an Admin deliberately imports or inspects one.
  const [governedCandidateId, setGovernedCandidateId] = useState("");
  const [generationRunId, setGenerationRunId] = useState(SYNTHETIC_GOVERNED_BUNDLE.generationRunId);
  const [governedRun, setGovernedRun] = useState<GovernedGenerationRunView | null>(null);
  const [mappingRationale, setMappingRationale] = useState(SYNTHETIC_EDITED_RATIONALE);
  const mappingRationaleDirty = useRef(false);
  const [governedChangeReason, setGovernedChangeReason] = useState("Apply the single controlled synthetic alternative.");
  const [validationIssues, setValidationIssues] = useState<GovernedValidationIssue[]>([]);
  const [commandError, setCommandError] = useState<string | null>(null);
  const [sourceActivationReceipt, setSourceActivationReceipt] = useState<string | null>(null);
  const [intakeBatch] = useState<ChecklistImportBatchView | null>(null);
  const templateLoad = useAdminLoad(() => backend.getTemplate({ templateId: "TPL-CABIN-2026" }), [backend]);
  const questionLoad = useAdminLoad(() => backend.listQuestions({}), [backend]);
  const regulatoryLoad = useAdminLoad(() => backend.listRegulatoryReferences({ status: "ACTIVE" }), [backend]);
  const governedLoad = useAdminLoad(
    () => governedCandidateId
      ? backend.getGovernedCandidate({ candidateId: governedCandidateId })
      : Promise.resolve(null),
    [backend, governedCandidateId],
  );
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
  const hasSourceGap = governedLoad.data?.questions.some((question) =>
    question.regulatoryTrace.state === "SOURCE_MAPPING_REQUIRED",
  ) ?? false;

  useEffect(() => {
    if (!mappingRationaleDirty.current && governedLoad.data?.mappings[0]?.rationale) {
      setMappingRationale(governedLoad.data.mappings[0].rationale);
    }
  }, [governedLoad.data]);

  async function importGovernedCandidate(
    candidateBundle: GovernedCandidateBundleInput,
    operationId: string,
    failureMessage: string,
  ) {
    setCommandError(null);
    setValidationIssues([]);
    try {
      const result = await backend.importGovernedGenerationRun({
        operationId,
        idempotencyKey: operationId,
        candidateBundle,
      });
      mappingRationaleDirty.current = false;
      if (result.candidate?.mappings[0]?.rationale) setMappingRationale(result.candidate.mappings[0].rationale);
      setGovernedRun(result);
      setGenerationRunId(result.generationRunId);
      if (result.candidate) setGovernedCandidateId(result.candidate.candidateId);
      governedLoad.reload();
    } catch (cause) {
      setCommandError(cause instanceof Error ? cause.message : failureMessage);
    }
  }

  async function importSyntheticGovernedCandidate() {
    return importGovernedCandidate(
      SYNTHETIC_GOVERNED_BUNDLE,
      "ADMIN-SYNTHETIC-GOVERNED-IMPORT",
      "Synthetic candidate import failed.",
    );
  }

  async function importLegacySourceGapCandidate() {
    return importGovernedCandidate(
      SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE,
      "ADMIN-SYNTHETIC-LEGACY-CANDIDATE-IMPORT",
      "Candidate-only legacy source-gap Draft import failed.",
    );
  }

  async function importCurrentSourceReconciliationCandidate() {
    const binding = SYNTHETIC_HYBRID_RECONCILED_BUNDLE.sourceCurrentness;
    if (!binding || !binding.previousSourceSnapshotId || !binding.previousSourceHash) {
      setCommandError("Current-source reconciliation fixture is missing its exact predecessor/current activation binding.");
      return;
    }
    setCommandError(null);
    setValidationIssues([]);
    try {
      const activation = await backend.activateGovernedSourceCurrentness({
        operationId: "ADMIN-SYNTHETIC-HYBRID-SOURCE-CURRENTNESS-ACTIVATION",
        idempotencyKey: "ADMIN-SYNTHETIC-HYBRID-SOURCE-CURRENTNESS-ACTIVATION",
        currentSourceSnapshotId: binding.currentSourceSnapshotId,
        currentSourceHash: binding.currentSourceHash,
        previousSourceSnapshotId: binding.previousSourceSnapshotId,
        previousSourceHash: binding.previousSourceHash,
        reason: "Activate the exact current synthetic source before creating a separate reconciliation Draft.",
      });
      setSourceActivationReceipt(`${activation.status} · ${activation.eventId} · ${activation.impactReviewDraftId ?? "no impact Draft"}`);
    } catch (cause) {
      setCommandError(cause instanceof Error ? cause.message : "Current-source activation failed.");
      return;
    }
    await importGovernedCandidate(
      SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
      "ADMIN-SYNTHETIC-HYBRID-RECONCILIATION-IMPORT",
      "Current-source reconciliation Draft import failed.",
    );
  }

  async function inspectGovernedGenerationRun() {
    setCommandError(null);
    setValidationIssues([]);
    try {
      const result = await backend.getGovernedGenerationRun({ generationRunId });
      mappingRationaleDirty.current = false;
      if (result.candidate?.mappings[0]?.rationale) setMappingRationale(result.candidate.mappings[0].rationale);
      setGovernedRun(result);
      if (result.candidate) setGovernedCandidateId(result.candidate.candidateId);
    } catch (cause) {
      setCommandError(cause instanceof Error ? cause.message : "Generation run inspection failed.");
    }
  }

  async function editGovernedCandidate() {
    if (!governedLoad.data) return;
    setCommandError(null);
    setValidationIssues([]);
    try {
      const mappings = governedLoad.data.mappings.map((mapping, index) => index === 0 ? { ...mapping, rationale: mappingRationale } : mapping);
      const successor = await backend.createGovernedCandidateRevision({
        operationId: `ADMIN-EDIT-${governedLoad.data.candidateId}-R${governedLoad.data.revision}`,
        idempotencyKey: `ADMIN-EDIT-${governedLoad.data.candidateId}-R${governedLoad.data.revision}`,
        candidateId: governedLoad.data.candidateId,
        expectedRevision: governedLoad.data.revision,
        expectedContentDigest: governedLoad.data.contentDigest,
        changeReason: governedChangeReason,
        mappings,
        questions: governedLoad.data.questions,
        requiredOwners: governedLoad.data.requiredOwners,
      });
      mappingRationaleDirty.current = false;
      if (successor.mappings[0]?.rationale) setMappingRationale(successor.mappings[0].rationale);
      setGovernedCandidateId(successor.candidateId);
      setGovernedRun((current) => current ? { ...current, candidate: successor } : current);
      governedLoad.reload();
    } catch (cause) {
      if (cause instanceof GovernedValidationError) setValidationIssues(cause.issues);
      else setCommandError(cause instanceof Error ? cause.message : "Candidate edit failed.");
    }
  }

  async function submitGovernedCandidate() {
    if (!governedLoad.data) return;
    setCommandError(null);
    setValidationIssues([]);
    try {
      const submitted = await backend.submitGovernedCandidateReview({
        operationId: `ADMIN-SUBMIT-${governedLoad.data.candidateId}-R${governedLoad.data.revision}`,
        idempotencyKey: `ADMIN-SUBMIT-${governedLoad.data.candidateId}-R${governedLoad.data.revision}`,
        candidateId: governedLoad.data.candidateId,
        expectedRevision: governedLoad.data.revision,
        expectedContentDigest: governedLoad.data.contentDigest,
        reason: "Submit exact candidate for department review.",
      });
      setGovernedRun((current) => current ? { ...current, candidate: submitted } : current);
      governedLoad.reload();
    } catch (cause) {
      setCommandError(cause instanceof Error ? cause.message : "Candidate submission failed.");
    }
  }

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
      <ChecklistIntakePanel
        batch={intakeBatch}
        disabledReason="The supplied AGA archive is an external, read-only dependency; an Admin must receive it through the governed intake route before candidate review."
      />
      <AGACandidateDemoPanel capability={adminBackend.agaCandidateDemo} />
      <section className="admin-template-identity" aria-label="Template identity">
        <div><span>Template master</span><b>TPL-CABIN-2026</b></div><div><span>Immutable published version</span><b>CTV-CABIN-1</b></div><div><span>Published owner</span><b>Department Manager</b></div>
      </section>
      <AdminError message={regulatoryLoad.error} />
      <section className="admin-record-card" aria-label="Governed candidate editor">
        <h2>Governed checklist candidate</h2>
        <AdminError message={governedLoad.error} />
        <div className="admin-builder-row-actions">
          <button type="button" onClick={() => void importSyntheticGovernedCandidate()}>Import synthetic governed candidate</button>
          <button type="button" onClick={() => void importLegacySourceGapCandidate()}>Load candidate-only legacy source-gap Draft</button>
        </div>
        <div className="admin-builder-row-actions">
          <label>
            Generation run ID
            <input
              aria-label="Generation run ID"
              value={generationRunId}
              onChange={(event) => setGenerationRunId(event.target.value)}
            />
          </label>
          <button type="button" onClick={() => void inspectGovernedGenerationRun()}>
            Inspect governed generation run
          </button>
        </div>
        {governedRun ? (
          <dl>
            <div><dt>Run / request</dt><dd>{governedRun.generationRunId} · {governedRun.requestId} · {governedRun.status}</dd></div>
            <div><dt>Versions</dt><dd>{governedRun.inputSchemaVersion} · {governedRun.generationPolicyVersion} · {governedRun.providerCatalogVersion} · {governedRun.providerAdapterVersion}</dd></div>
            <div><dt>Provider / target</dt><dd>{governedRun.providerId} · {governedRun.inspectionType} · {governedRun.targetId}</dd></div>
            <div><dt>Imported run input digest</dt><dd>{governedRun.inputDigest}</dd></div>
            <div><dt>Imported run output digest</dt><dd>{governedRun.outputDigest}</dd></div>
            {governedRun.failure ? <div><dt>Failure</dt><dd>{governedRun.failure.code} · {governedRun.failure.reason} · {governedRun.failure.operationId}</dd></div> : null}
          </dl>
        ) : null}
        {governedLoad.data ? (
          <>
          <dl>
            <div><dt>Current candidate content digest</dt><dd>{governedLoad.data.contentDigest}</dd></div>
          </dl>
          <p>{governedLoad.data.candidateId} · revision {governedLoad.data.revision} · {governedLoad.data.status} · {governedLoad.data.contentDigest}</p>
          <p><b>Change reason:</b> {governedLoad.data.changeReason}</p>
          <p>Lineage: root {governedLoad.data.candidateRootId} · supersedes {governedLoad.data.supersedesCandidateId ?? "none"} · run {governedLoad.data.generationRunId} · template {governedLoad.data.templateId} · schema {governedLoad.data.schemaVersion}</p>
          <p>Source snapshots: {governedLoad.data.sourceSnapshots.map((source) => String(source.locator ?? source.sourceId)).join(" · ")}</p>
          {sourceActivationReceipt ? <p><b>Current-source activation:</b> {sourceActivationReceipt}</p> : null}
          <p>Scope facts: {governedLoad.data.scopeFactIds.join(" · ")} · input partitions: {governedLoad.data.crosswalkPartitionIds.join(" · ")}</p>
          <p>Required owners: {governedLoad.data.requiredOwners.map((owner) => `${String(owner.departmentId)}/${String(owner.organizationalUnitId)}`).join(" · ")}</p>
          {validationIssues.length > 0 ? <div role="alert">{validationIssues.map((issue) => <p key={`${issue.fieldPath}-${issue.code}`}>{issue.fieldPath}: {issue.message} · {issue.sourceIdentity} · {issue.clauseId} · {issue.locator}</p>)}</div> : null}
          {governedLoad.data.mappings.map((mapping, index) => <details key={String(mapping.mappingId)} open><summary>{String(mapping.mappingId)} · {String(mapping.requirement)}</summary>{index === 0 && governedLoad.data?.status === "GENERATED_DRAFT" && !hasSourceGap ? <label>Mapping rationale<textarea aria-label="Mapping rationale" value={mappingRationale} onChange={(event) => { mappingRationaleDirty.current = true; setMappingRationale(event.target.value); }} /></label> : <p><b>Rationale:</b> {String(mapping.rationale)}</p>}<p><b>Relationship / applicability:</b> {mapping.relationship} · {mapping.applicability}</p><p><b>Citations:</b> {JSON.stringify(mapping.citations)}</p></details>)}
          {governedLoad.data.questions.map((question) => <GovernedQuestionGovernance key={String(question.questionId)} question={question} />)}
          {governedLoad.data.status === "GENERATED_DRAFT" ? hasSourceGap ? (
            <aside className="admin-source-gap" aria-label="Source-gap repair path">
              <b>Repair required before review</b>
              <p>This candidate-only legacy Draft has no authority. Activate the exact current source first, then create a separate current-source reconciliation Draft; the legacy version remains immutable candidate input.</p>
              <div className="admin-builder-row-actions">
                <button type="button" onClick={() => void importCurrentSourceReconciliationCandidate()}>Activate source change and create reconciliation Draft</button>
                <button disabled title="SOURCE_MAPPING_REQUIRED must be resolved in a separate current-source reconciliation Draft before an immutable edit can be created." type="button">Create immutable edited revision</button>
                <button disabled title="SOURCE_MAPPING_REQUIRED blocks department review, technical approval, publication, and executable Audit-package use." type="button">Submit exact candidate for department review</button>
              </div>
            </aside>
          ) : <div className="admin-builder-row-actions"><label>Change reason<textarea aria-label="Change reason" value={governedChangeReason} onChange={(event) => setGovernedChangeReason(event.target.value)} /></label><button type="button" onClick={() => void editGovernedCandidate()}>Create immutable edited revision</button><button type="button" onClick={() => void submitGovernedCandidate()}>Submit exact candidate for department review</button></div> : <p>Submitted for department review. Admin has no technical approval or publication action.</p>}
          </>
        ) : null}
      </section>
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
