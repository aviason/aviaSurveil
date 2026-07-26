import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AssistantDraftView, FindingView } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

function assistantDisplayFinding(source: FindingView): FindingView {
  if (source.id !== "FND-CAB-2026-001") return source;
  return {
    ...source,
    findingNumber: "CAB-2026-011",
    title: "Emergency equipment serviceability record incomplete",
    description: "The sampled emergency equipment position did not have a complete serviceability record available for review.",
    regulatoryReference: "Configured reference CAB-EME-01 (demo)",
  };
}

export function InspectorAssistantPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("inspector") ?? runtime.backend, [runtime]);
  const navigate = useNavigate();
  const [finding, setFinding] = useState<FindingView | null>(null);
  const [prompt, setPrompt] = useState("Draft source-referenced review guidance for this Finding.");
  const [draft, setDraft] = useState<AssistantDraftView | null>(null);
  const [reviewResults, setReviewResults] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (!backend.assistantDrafts) return;
    let cancelled = false;
    void Promise.all([backend.assistantDrafts.getGuidance({}), backend.findings.list({ limit: 50 })]).then(([, page]) => {
      if (!cancelled) setFinding(page.items.find((item) => item.id === "FND-CAB-2026-001") ?? page.items[0] ?? null);
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);
  async function createDraft() {
    if (!backend.assistantDrafts || !finding) return;
    try { setDraft(await backend.assistantDrafts.createDraft({ findingId: finding.id, prompt })); }
    catch (cause) { setError(errorMessage(cause)); }
  }
  const displayFinding = finding ? assistantDisplayFinding(finding) : null;
  const suggestions = displayFinding?.id === "FND-CAB-2026-001" ? [
    { id: "finding-language", title: "Draft finding language for PBE serviceability", source: "Sources: REG-CABIN-EQ-2026 / CAB EMEQ PBE · Checklist item cab-em-eq-pbe", text: "The inspected PBE position could not be confirmed as serviceable and accessible against the configured cabin emergency equipment check." },
    { id: "checklist-focus", title: "Suggested checklist focus", source: "Sources: Risk profile RISK-ORG-XYZ-2026 · NAMCARS CAB EMEQ PBE", text: "Add targeted sampling of PBE serviceability and compare CAP effectiveness against accepted emergency equipment closure evidence." },
  ] : displayFinding ? [
    { id: "finding-language", title: `Draft finding language for ${displayFinding.findingNumber}`, source: `Sources: ${displayFinding.regulatoryReference ?? displayFinding.findingBasis} · ${displayFinding.organizationName}`, text: `Draft review language for ${displayFinding.findingNumber}, “${displayFinding.title}”: ${displayFinding.description}` },
    { id: "checklist-focus", title: `Suggested review focus for ${displayFinding.findingNumber}`, source: `Sources: Finding basis for ${displayFinding.findingNumber} · ${displayFinding.organizationName}`, text: `Review the configured basis for “${displayFinding.title}” and request only the expected evidence relevant to this Finding.` },
  ] : [];
  return <WorkspaceShell roleLabel="CAA Inspector" routeLabel="AI Inspector Assistant">
    <div className="inspector-secondary-page inspector-assistant-page" data-testid="inspector-assistant-page">
      <header className="inspector-secondary-head workbench-page-header"><div><h1>AI Inspector Assistant Panel</h1><p>Review source-referenced draft assistance with accept, edit and reject controls.</p></div><button className="inspector-secondary-button" onClick={() => navigate(finding?.id === "FND-CAB-2026-001" ? "/inspector/findings/FND-CAB-2026-001" : "/inspector/findings")} type="button">{finding?.id === "FND-CAB-2026-001" ? "Back to Finding" : "Back to Findings"}</button></header>
      <div className="inspector-draft-guardrails"><span>AI-generated Draft - requires authorized review</span><span>Demo data</span><span>Not a legal decision</span><span>No real AI service</span></div>
      <CommandError message={error} />
      {displayFinding ? <section className="inspector-assistant-context" data-finding-id={finding?.id}><div><small>Finding context</small><h2>{displayFinding.findingNumber} · {displayFinding.title}</h2><p>{displayFinding.description}</p></div><dl><div><dt>Organization</dt><dd>{displayFinding.organizationName}</dd></div><div><dt>Status</dt><dd>{displayFinding.status.replaceAll("_", " ")}</dd></div><div><dt>Configured reference</dt><dd>{displayFinding.regulatoryReference ?? "Configured reference"}</dd></div></dl></section> : null}
      {displayFinding ? <section className="inspector-suggestion-register" aria-label="Assistant suggestions">
        <div className="inspector-suggestion-register__head" aria-hidden="true"><span>Draft suggestion</span><span>Status</span><span>Review text</span><span>Authorized review</span></div>
        {suggestions.map((suggestion) => <article aria-label={suggestion.title} className="inspector-suggestion-row" key={suggestion.id}>
          <div className="inspector-suggestion-row__title"><h2>{suggestion.title}</h2><p>{suggestion.source}</p></div>
          <span className="inspector-suggestion-row__status">● Pending Review</span>
          <div className="inspector-suggestion-row__text"><textarea aria-label={`${suggestion.title} review text`} defaultValue={suggestion.text} /><p>AI-generated draft - requires authorized review. The Inspector must verify facts and wording before use. Not a legal decision.</p></div>
          <div className="inspector-suggestion-row__actions"><button aria-label={`Accept draft: ${suggestion.title}`} aria-pressed={reviewResults[suggestion.id] === "Accepted by authorized Inspector"} onClick={() => setReviewResults((current) => ({ ...current, [suggestion.id]: "Accepted by authorized Inspector" }))} type="button">Accept draft</button><button aria-label={`Record edit: ${suggestion.title}`} aria-pressed={reviewResults[suggestion.id] === "Inspector edit recorded"} onClick={() => setReviewResults((current) => ({ ...current, [suggestion.id]: "Inspector edit recorded" }))} type="button">Record edit</button><button aria-label={`Reject: ${suggestion.title}`} aria-pressed={reviewResults[suggestion.id] === "Draft rejected"} onClick={() => setReviewResults((current) => ({ ...current, [suggestion.id]: "Draft rejected" }))} type="button">Reject</button>{reviewResults[suggestion.id] ? <span role="status">{reviewResults[suggestion.id]}</span> : null}</div>
        </article>)}
      </section> : null}
      <section className="inspector-draft-request"><label>Draft request<textarea value={prompt} onChange={(event) => setPrompt(event.target.value)} /></label><button className="inspector-secondary-button inspector-secondary-button--primary" disabled={!finding} onClick={() => void createDraft()} type="button">Create Draft</button></section>
      {draft ? <section aria-label="Assistant Draft" className="inspector-draft-output" data-durable-outcome="assistant-draft" role="status"><span>Draft · advisory only</span><h2>Draft review guidance</h2><p>{draft.draft}</p><small>This Draft cannot create a Finding, set severity, close a Finding, or take enforcement action.</small></section> : null}
    </div>
  </WorkspaceShell>;
}
