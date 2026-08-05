import { useEffect, useState, type ReactNode } from "react";

import type {
  AGACandidateDemoBackend,
  AGACandidateDemoForm,
  AGACandidateDemoQuestion,
  AGACandidateDemoSummary,
} from "../../backend/aga-candidate-demo";

type PanelState =
  | { kind: "checking" }
  | { kind: "unavailable" }
  | { kind: "available"; summary: AGACandidateDemoSummary | null; forms: AGACandidateDemoForm[]; questions: AGACandidateDemoQuestion[]; error: string | null };

function copyForSourceGap(question: AGACandidateDemoQuestion) {
  return question.sourceGapCategory === "UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL"
    ? "No question-level source proposal is present. This candidate cannot establish source authority."
    : "Source mapping remains required; the candidate proposal is not an attestation or approval.";
}

async function loadAllQuestions(
  capability: AGACandidateDemoBackend,
  options: { signal: AbortSignal },
): Promise<AGACandidateDemoQuestion[]> {
  const questions: AGACandidateDemoQuestion[] = [];
  const seenCursors = new Set<string>();
  let cursor: string | undefined;
  for (;;) {
    const page = await capability.listQuestions({ cursor, limit: 100 }, options);
    questions.push(...page.items);
    if (!page.nextCursor) return questions;
    if (seenCursors.has(page.nextCursor) || questions.length >= 5000) {
      throw new Error("The sealed candidate question pagination did not converge.");
    }
    seenCursors.add(page.nextCursor);
    cursor = page.nextCursor;
  }
}

/**
 * Intentionally self-contained read view. It has no mutation, persistence,
 * analytics, or mock path; the server capability decides whether it renders.
 */
export function AGACandidateDemoPanel({ capability, supplementalLink }: { capability: AGACandidateDemoBackend | undefined; supplementalLink?: ReactNode }) {
  const [state, setState] = useState<PanelState>({ kind: "checking" });

  useEffect(() => {
    const controller = new AbortController();
    const invalidateRestoredPage = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      controller.abort();
      setState({ kind: "unavailable" });
    };
    window.addEventListener("pageshow", invalidateRestoredPage);
    if (!capability) {
      setState({ kind: "unavailable" });
      return () => {
        controller.abort();
        window.removeEventListener("pageshow", invalidateRestoredPage);
      };
    }
    setState({ kind: "checking" });
    void capability.capability({}, { signal: controller.signal })
      .then(async (result) => {
        if (!result.available || controller.signal.aborted) {
          setState({ kind: "unavailable" });
          return;
        }
        try {
          const [summary, forms, questions] = await Promise.all([
            capability.summary({}, { signal: controller.signal }),
            capability.listForms({ limit: 100 }, { signal: controller.signal }),
            loadAllQuestions(capability, { signal: controller.signal }),
          ]);
          if (!controller.signal.aborted) {
            setState({ kind: "available", summary, forms: forms.items, questions, error: null });
          }
        } catch (cause) {
          if (!controller.signal.aborted) {
            setState({
              kind: "available",
              summary: null,
              forms: [],
              questions: [],
              error: cause instanceof Error ? cause.message : "The sealed candidate projection could not be read.",
            });
          }
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) setState({ kind: "unavailable" });
      });
    return () => {
      controller.abort();
      window.removeEventListener("pageshow", invalidateRestoredPage);
    };
  }, [capability]);

  if (state.kind === "checking" || state.kind === "unavailable") return null;
  const distinctRows = [...new Map(state.questions.map((question) => [question.proposalId, question])).values()];
  const proposalRows = distinctRows.filter((question) => question.sourceGapCategory === "PROPOSAL_PRESENT_REVIEW_REQUIRED");
  const unmappedRows = distinctRows.filter((question) => question.sourceGapCategory === "UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL");
  const blockerRows = distinctRows.filter((question) => question.riskBand === "PROPOSED_REVIEW_REQUIRED");

  return (
    <section className="aga-candidate-demo" aria-labelledby="aga-candidate-demo-title" data-testid="aga-candidate-demo-panel">
      <header>
        <div>
          <p className="eyebrow">Disposable local-preprod overlay</p>
          <h2 id="aga-candidate-demo-title">AGA candidate demo</h2>
          <p>This is a sealed, read-only candidate projection. It creates no governed record, decision, publication, Audit, or Finding.</p>
        </div>
        <div className="aga-candidate-demo__labels" aria-label="Candidate demo status">
          {(state.summary?.labels ?? []).map((label) => <span key={label}>{label}</span>)}
        </div>
        {supplementalLink ? <div className="aga-candidate-demo__supplemental-link">{supplementalLink}</div> : null}
      </header>
      {state.error ? <p className="command-error" role="status">{state.error}</p> : null}
      {state.summary ? (
        <dl className="aga-candidate-demo__summary">
          <div><dt>Sealed package digest</dt><dd>{state.summary.packageDigest}</dd></div>
          <div><dt>Forms</dt><dd>{state.summary.formCount}</dd></div>
          <div><dt>Candidate questions</dt><dd>{state.summary.questionCount}</dd></div>
        </dl>
      ) : null}
      <section aria-labelledby="aga-candidate-forms-title">
        <h3 id="aga-candidate-forms-title">All forms</h3>
        <p className="aga-candidate-demo__caption">First sealed page of forms. Question extraction state is descriptive only; it is not an import command.</p>
        <div className="aga-candidate-demo__table-wrap">
          <table>
            <thead><tr><th scope="col">Form</th><th scope="col">Title</th><th scope="col">Questions</th><th scope="col">Extraction state</th></tr></thead>
            <tbody>{state.forms.map((form) => <tr key={form.code}><th scope="row">{form.code}</th><td>{form.title}</td><td>{form.questionCount}</td><td>{form.questionExtractionState}</td></tr>)}</tbody>
          </table>
        </div>
      </section>
      <section aria-labelledby="aga-candidate-extraction-title">
        <h3 id="aga-candidate-extraction-title">Extraction review</h3>
        <p className="aga-candidate-demo__caption">Forms without a protocol-question boundary remain candidate-only and require human review.</p>
        <div className="aga-candidate-demo__table-wrap"><table><thead><tr><th scope="col">Form</th><th scope="col">Extraction state</th><th scope="col">Candidate questions</th></tr></thead><tbody>{state.forms.filter((form) => form.questionExtractionState === "NO_PROTOCOL_QUESTION_BOUNDARY_DETECTED").map((form) => <tr key={form.code}><th scope="row">{form.code}</th><td>{form.questionExtractionState}</td><td>{form.questionCount}</td></tr>)}</tbody></table></div>
      </section>
      <section aria-labelledby="aga-candidate-source-title">
        <h3 id="aga-candidate-source-title">Source-gap review</h3>
        <p className="aga-candidate-demo__caption">Synthetic Department Manager demo handoff · read-only candidate questions. No real Department Manager identity, assignment, decision, or publication record exists.</p>
        <ul className="aga-candidate-demo__requirements" aria-label="Future source evidence requirements">{(state.summary?.sourceRequirements ?? []).map((requirement) => <li key={requirement}>{requirement.replaceAll("_", " ")}</li>)}</ul>
        <div className="aga-candidate-demo__table-wrap" data-testid="aga-candidate-manager-demo-handoff"><table><caption className="sr-only">Synthetic Department Manager demo handoff</caption><thead><tr><th scope="col">Proposal</th><th scope="col">Form / ordinal</th><th scope="col">Candidate question</th><th scope="col">Source state</th><th scope="col">Boundary</th></tr></thead><tbody>{distinctRows.map((question) => <tr data-testid="aga-candidate-manager-question-row" key={question.proposalId}><th scope="row">{question.proposalId}</th><td>{question.formCode} · {question.ordinal}</td><td>{question.text}</td><td>SOURCE_MAPPING_REQUIRED</td><td>{copyForSourceGap(question)}</td></tr>)}</tbody></table></div>
      </section>
      <section aria-labelledby="aga-candidate-risk-title">
        <h3 id="aga-candidate-risk-title">Risk-review blockers</h3>
        <p className="aga-candidate-demo__caption">Every band is provisional. No approved risk, safety-critical decision, or Finding severity is available here.</p>
        <div className="aga-candidate-demo__table-wrap"><table><thead><tr><th scope="col">Proposal</th><th scope="col">Form / ordinal</th><th scope="col">Provisional risk band</th><th scope="col">Review state</th></tr></thead><tbody>{blockerRows.map((question) => <tr key={question.proposalId}><th scope="row">{question.proposalId}</th><td>{question.formCode} · {question.ordinal}</td><td>Provisional · {question.riskBand}</td><td>Expert review blocker</td></tr>)}</tbody></table></div>
      </section>
    </section>
  );
}
