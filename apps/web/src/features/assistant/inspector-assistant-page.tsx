import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AssistantDraftView, FindingView } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

export function InspectorAssistantPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("inspector") ?? runtime.backend, [runtime]);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [finding, setFinding] = useState<FindingView | null>(null);
  const [prompt, setPrompt] = useState("Draft source-referenced review guidance for this Finding.");
  const [draft, setDraft] = useState<AssistantDraftView | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (!backend.assistantDrafts) return;
    let cancelled = false;
    const selectedFindingId = searchParams.get("findingId");
    void Promise.all([backend.assistantDrafts.getGuidance({}), backend.findings.list({ limit: 50 })]).then(([, page]) => {
      if (!cancelled) setFinding(selectedFindingId ? page.items.find((item) => item.id === selectedFindingId) ?? null : null);
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, searchParams]);
  async function createDraft() {
    if (!backend.assistantDrafts || !finding) return;
    try { setDraft(await backend.assistantDrafts.createDraft({ findingId: finding.id, prompt })); }
    catch (cause) { setError(errorMessage(cause)); }
  }
  const displayFinding = finding;
  const suggestions = displayFinding ? [
    { id: "finding-language", title: `Draft finding language for ${displayFinding.findingNumber}`, source: `Sources: ${displayFinding.regulatoryReference ?? displayFinding.findingBasis} · ${displayFinding.organizationName}`, text: `Draft review language for ${displayFinding.findingNumber}, “${displayFinding.title}”: ${displayFinding.description}` },
    { id: "checklist-focus", title: `Suggested review focus for ${displayFinding.findingNumber}`, source: `Sources: Finding basis for ${displayFinding.findingNumber} · ${displayFinding.organizationName}`, text: `Review the configured basis for “${displayFinding.title}” and request only the expected evidence relevant to this Finding.` },
  ] : [];
  return <WorkspaceShell roleLabel="CAA Inspector" routeLabel="AI Inspector Assistant">
    <div className="inspector-secondary-page inspector-assistant-page" data-testid="inspector-assistant-page">
      <header className="inspector-secondary-head workbench-page-header"><div><h1>AI Inspector Assistant Panel</h1><p>Review source-referenced advisory drafts before using any wording in an authorized workflow.</p></div><button className="inspector-secondary-button" onClick={() => navigate(finding ? `/inspector/findings/${encodeURIComponent(finding.id)}` : "/inspector/findings")} type="button">{finding ? "Back to Finding" : "Back to Findings"}</button></header>
      <div className="inspector-draft-guardrails"><span>AI-generated draft · requires authorized review</span><span>Advisory only</span><span>Not a legal decision</span></div>
      <CommandError message={error} />
      {displayFinding ? <section className="inspector-assistant-context" data-finding-id={finding?.id}><div><small>Finding context</small><h2>{displayFinding.findingNumber} · {displayFinding.title}</h2><p>{displayFinding.description}</p></div><dl><div><dt>Organization</dt><dd>{displayFinding.organizationName}</dd></div><div><dt>Status</dt><dd>{displayFinding.status.replaceAll("_", " ")}</dd></div><div><dt>Configured reference</dt><dd>{displayFinding.regulatoryReference ?? "Configured reference"}</dd></div></dl></section> : null}
      {displayFinding ? <section className="inspector-suggestion-register" aria-label="Assistant suggestions">
        <div className="inspector-suggestion-register__head" aria-hidden="true"><span>Draft suggestion</span><span>Status</span><span>Review text</span><span>Server boundary</span></div>
        {suggestions.map((suggestion) => <article aria-label={suggestion.title} className="inspector-suggestion-row" key={suggestion.id}>
          <div className="inspector-suggestion-row__title"><h2>{suggestion.title}</h2><p>{suggestion.source}</p></div>
          <span className="inspector-suggestion-row__status">● Pending Review</span>
          <div className="inspector-suggestion-row__text"><textarea aria-label={`${suggestion.title} review text`} defaultValue={suggestion.text} /><p>AI-generated draft - requires authorized review. The Inspector must verify facts and wording before use. Not a legal decision.</p></div>
          <div className="inspector-suggestion-row__actions"><span role="note">No local review mutation. Use the declared Finding and report commands for workflow decisions.</span></div>
        </article>)}
      </section> : null}
      <section className="inspector-draft-request"><label>Draft request<textarea value={prompt} onChange={(event) => setPrompt(event.target.value)} /></label><button className="inspector-secondary-button inspector-secondary-button--primary" disabled={!finding} onClick={() => void createDraft()} type="button">Create Draft</button></section>
      {draft ? <section aria-label="Assistant Draft" className="inspector-draft-output" data-durable-outcome="assistant-draft" role="status"><span>Draft · advisory only</span><h2>Draft review guidance</h2><p>{draft.draft}</p><small>This Draft cannot create a Finding, set severity, close a Finding, or take enforcement action.</small></section> : null}
    </div>
  </WorkspaceShell>;
}
