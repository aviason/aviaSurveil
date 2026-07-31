import { useState } from "react";

import { AdminError, AdminGuardrails, AdminPage, useAdminLoad, useAdminWorkspace } from "./admin-workspace-shared";

const sourceTypeLabels = {
  ICAO_PQ: "ICAO PQ",
  ANNEX_CROSSWALK: "Annex / national crosswalk",
  AUDIT_AREA_TAXONOMY: "Audit-area taxonomy",
  NATIONAL_LIBRARY: "National regulatory library",
  CAA_PROCEDURE: "CAA procedure",
} as const;

export function RegulatoryLibraryPage() {
  const backend = useAdminWorkspace();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [auditArea, setAuditArea] = useState("");
  const [serviceProvider, setServiceProvider] = useState("");
  const { data, error } = useAdminLoad(() => backend.listRegulatoryReferences({ search, status }), [backend, search, status]);
  const governedSources = useAdminLoad(() => backend.listGovernedSources({}), [backend]);
  const mappings = data?.items.flatMap((reference) => reference.mappings) ?? [];
  const auditAreas = [...new Set(mappings.map((mapping) => mapping.auditArea))].sort();
  const serviceProviders = [...new Set(mappings.flatMap((mapping) => mapping.serviceProviderTypes))].sort();
  const references = data?.items
    .map((reference) => ({
      ...reference,
      mappings: reference.mappings.filter(
        (mapping) =>
          (!auditArea || mapping.auditArea === auditArea) &&
          (!serviceProvider || mapping.serviceProviderTypes.includes(serviceProvider)),
      ),
    }))
    .filter((reference) => (!auditArea && !serviceProvider) || reference.mappings.length > 0);

  return (
    <AdminPage testId="admin-regulatory-library-page" routeLabel="Regulatory Library" title="Regulatory Library" description="NAMCARS Library and Regulatory Cross-Reference share this read-only configured-reference surface.">
      <AdminGuardrails />
      <section className="admin-filter-bar" aria-label="Regulatory Library filters">
        <label>Search<input aria-label="Search regulatory references" onChange={(event) => setSearch(event.target.value)} value={search} /></label>
        <label>Status<select aria-label="Regulatory status" onChange={(event) => setStatus(event.target.value)} value={status}><option value="">All statuses</option><option value="ACTIVE">Active</option><option value="SUPERSEDED">Superseded</option></select></label>
        <label>Audit area<select aria-label="Regulatory audit area" onChange={(event) => setAuditArea(event.target.value)} value={auditArea}><option value="">All audit areas</option>{auditAreas.map((area) => <option key={area} value={area}>{area}</option>)}</select></label>
        <label>Service provider<select aria-label="Regulatory service provider" onChange={(event) => setServiceProvider(event.target.value)} value={serviceProvider}><option value="">All service providers</option>{serviceProviders.map((provider) => <option key={provider} value={provider}>{provider}</option>)}</select></label>
      </section>
      <AdminError message={error} />
      <AdminError message={governedSources.error} />
      <section className="admin-record-card" aria-label="Governed source snapshots">
        <h2>Governed source snapshots</h2>
        <p>Exact source identity, hash, locator, partition lineage, and affected candidate are rendered from the candidate-only governed boundary.</p>
        <ul>
          {governedSources.data?.items.map((source) => (
            <li key={`${source.sourceId}-${source.clauseId}`}>
              <b>{source.title}</b>
              <small>
                {source.sourceIdentity} · {source.versionIdentity} · {source.sourceHash}
              </small>
              <p>{source.locator} · {source.clauseLocator}</p>
              <p>
                Partitions:{" "}
                {source.partitions
                  .map(
                    (partition) =>
                      `${partition.role} · ${partition.partitionId} · ${partition.stableRowIdentity}`,
                  )
                  .join(" | ") || "none"}
              </p>
              <p>
                Applicability:{" "}
                {source.applicabilityFacts
                  .map(
                    (fact) =>
                      `${fact.mappingId}: ${fact.relationship}/${fact.applicability}${
                        fact.sourceGap ? ` · ${fact.sourceGap}` : ""
                      }`,
                  )
                  .join(" | ") || "No persisted applicability decision"}
              </p>
              <p>
                generation runs: {source.generationRunIds.join(", ") || "none"} ·
                affected candidates: {source.candidateIds.join(", ") || "none"} ·
                unresolved gaps:{" "}
                {source.unresolvedGaps
                  .map((gap) => `${gap.gapId}: ${gap.reason}`)
                  .join(" | ") || "none"}
              </p>
            </li>
          ))}
        </ul>
      </section>
      <div className="admin-mapping-register" role="list" aria-label="Configured regulatory references">
        {references?.map((reference) => (
          <article className="admin-record-card" key={reference.id} role="listitem">
            <header><div><b>{reference.title}</b><small>{reference.id}</small></div><span>{reference.status}</span></header>
            <dl><div><dt>Version</dt><dd>{reference.version}</dd></div><div><dt>Effective date</dt><dd>{reference.effectiveDate}</dd></div></dl>
            <h2>Configured rules</h2><ul>{reference.configuredRules.map((rule) => <li key={rule}>{rule}</li>)}</ul>
            <h2>Change history</h2><ul>{reference.changeHistory.map((change) => <li key={change}>{change}</li>)}</ul>
            {reference.mappings.map((mapping) => (
              <section className="admin-regulatory-mapping" key={mapping.id} aria-labelledby={`${mapping.id}-title`}>
                <header>
                  <div>
                    <p className="eyebrow">{mapping.auditArea} · {mapping.serviceProviderTypes.join(", ")}</p>
                    <h2 id={`${mapping.id}-title`}>Requirements-to-inspection pilot</h2>
                    <small>{mapping.id}</small>
                  </div>
                  <span className={`admin-review-badge admin-review-badge--${mapping.reviewStatus.toLowerCase()}`}>{mapping.reviewStatus.replaceAll("_", " ")}</span>
                </header>
                <p className="admin-regulatory-scope"><b>Applicable regulation scope:</b> {mapping.applicableRegulations.join(" · ")}</p>
                <div className="admin-trace-chain" aria-label={`Trace chain for ${mapping.id}`}>
                  <section><span>1 · ICAO starting point</span><b>PQ {mapping.protocolQuestionId} · {mapping.criticalElement}</b><p>{mapping.protocolQuestion}</p></section>
                  <section><span>2 · Annex / SARP</span><ul>{mapping.annexReferences.map((reference) => <li key={reference}>{reference}</li>)}</ul></section>
                  <section><span>3 · National regulation</span><ul>{mapping.nationalReferences.map((reference) => <li key={reference}>{reference}</li>)}</ul></section>
                  <section><span>4 · CAA implementation</span><p>{mapping.caaImplementationReference}</p></section>
                  <section><span>5 · Requirement</span><p>{mapping.requirement}</p></section>
                  <section><span>6 · What must be verified</span><p>{mapping.verificationObjective}</p></section>
                </div>
                {mapping.sourceGap ? <aside className="admin-source-gap"><b>Controlled-source gap</b><p>{mapping.sourceGap}</p></aside> : null}
                <section className="admin-refresh-policy" aria-labelledby={`${mapping.id}-refresh-title`}>
                  <header>
                    <div>
                      <p className="eyebrow">Regulatory source governance</p>
                      <h3 id={`${mapping.id}-refresh-title`}>Event-driven review with scheduled reconciliation</h3>
                    </div>
                    <span>{mapping.refreshPolicy.sourceChangeState.replaceAll("_", " ")}</span>
                  </header>
                  <dl>
                    <div><dt>Captured source set</dt><dd>{mapping.refreshPolicy.documentCount} public files · {mapping.refreshPolicy.sourceCollectionId}</dd></div>
                    <div><dt>Last source check</dt><dd>{mapping.refreshPolicy.lastCheckedAt}</dd></div>
                    <div><dt>Next reconciliation</dt><dd>{mapping.refreshPolicy.nextReconciliationDate} · every {mapping.refreshPolicy.reconciliationIntervalMonths} months</dd></div>
                    <div><dt>Next expert validation</dt><dd>{mapping.refreshPolicy.nextExpertValidationDate} · every {mapping.refreshPolicy.expertValidationIntervalMonths} months</dd></div>
                    <div><dt>Change handling</dt><dd>{mapping.refreshPolicy.updateMode.replaceAll("_", " ")}</dd></div>
                    <div><dt>Tracked manifest</dt><dd>{mapping.refreshPolicy.manifestPath}</dd></div>
                  </dl>
                  <ul>{mapping.refreshPolicy.guardrails.map((guardrail) => <li key={guardrail}>{guardrail}</li>)}</ul>
                </section>
                <div className="admin-regulatory-detail-grid">
                  <section><h3>Why this mapping is included</h3><p>{mapping.whyIncluded}</p></section>
                  <section><h3>Expected evidence</h3><ul>{mapping.expectedEvidence.map((evidence) => <li key={evidence}>{evidence}</li>)}</ul></section>
                </div>
                <h3>Candidate inspection questions</h3>
                <ol className="admin-proposed-questions">
                  {mapping.proposedQuestions.map((question) => (
                    <li key={question.id}>
                      <div><b>{question.prompt}</b><small>{question.id}</small></div>
                      <p><b>Verify:</b> {question.verificationMethod}</p>
                      <p><b>Evidence:</b> {question.evidenceExamples.join(" · ")}</p>
                      <p><b>Why included:</b> {question.whyIncluded}</p>
                    </li>
                  ))}
                </ol>
                <h3>Controlled sources</h3>
                <ul className="admin-regulatory-sources">
                  {mapping.sources.map((source) => (
                    <li key={source.id}>
                      <div><b>{source.url ? <a href={source.url} rel="noreferrer" target="_blank">{source.title}</a> : source.title}</b><span>{sourceTypeLabels[source.sourceType]} · {source.status.replaceAll("_", " ")}</span></div>
                      <small>{source.version} · {source.locator}</small>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
            <p className="admin-regulatory-disclaimer">Candidate-only configured reference. Expert validation remains mandatory; this is not legal advice, an official compliance determination, or authority to publish a checklist.</p>
          </article>
        ))}
      </div>
    </AdminPage>
  );
}
