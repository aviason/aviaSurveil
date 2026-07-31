# Audit Checklist Workflow

## Purpose

Help inspectors run an exact Audit checklist and submit reviewable Potential
Findings from Non-Compliant or Observation results.

## Checklist development upstream

Checklist development follows a requirements-to-inspection sequence:

1. Start from the applicable ICAO Protocol Question and Critical Element.
2. Identify the exact Annex/SARP source and version.
3. Map the SARP to the applicable national regulation.
4. Identify the CAA procedure, manual, guidance, tool, or process that
   implements the requirement.
5. Interpret the regulatory requirement and its intended outcome.
6. State what the inspector must verify in the scoped inspection.
7. Decompose that verification objective into one or more practical inspection
   questions.
8. Define the Evidence needed for each question.
9. Have the currently assigned responsible Department Manager technically
   review the source chain, applicability, interpretation, questions, and
   Evidence within the department scope.
10. Record a separate Department Manager publication decision before an
    approved question enters an immutable versioned checklist.

The inspector ultimately runs a practical checklist, not the ICAO PQ. Each
question nevertheless remains traceable through its exact regulatory mapping.
The mapping graph is not copied into every response.

## Regulatory source refresh

1. Synchronize only the configured public source collection and record the
   exact URL, observed page metadata, byte size, and SHA-256 for each file.
2. Keep source bytes and extracted text in the controlled source vault while
   keeping the tracked manifest free of full regulatory content.
3. Reconcile the configured collection every six months and initiate review
   earlier when an authoritative change notification or source hash change is
   observed.
4. Compare source-set, file-identity, and content-hash changes. Extract
   candidate clause differences where reliable, retaining an explicit
   extraction limitation when a document is image-heavy or structurally
   ambiguous.
5. Map changed clauses to affected RegulatoryMappings, requirements, proposed
   questions, Evidence expectations, and service-provider scopes.
6. Present the impact set to the responsible Department Manager. Download or
   extraction success does not satisfy this validation gate.
7. An observed source version is inert until an Admin records an explicit,
   append-only source-currentness activation with the exact predecessor and
   current source snapshot/hash. A source transition creates an impact-review
   Draft; activation is neither a legal interpretation, technical approval, nor
   a publication decision.
8. Import an existing or historical checklist only as an
   `EXISTING_CHECKLIST_CANDIDATE`. Preserve its wording, operational intent,
   and result history as candidate input, then reconcile it to the current
   regulatory/controlled-CAA-procedure chain in a new immutable Draft.
9. Put accepted changes into a new checklist Draft. Keep published checklist
   versions and in-progress Audits immutable.
10. Perform a complete Department Manager technical review at least annually even if the
   six-month reconciliation found no byte changes.

## Adaptive inspection scope

Before assembling an inspection package, the system may produce an advisory
scope recommendation from exact, time-bounded inputs:

- open, overdue, repeat, and recently closed Findings;
- CAP and Evidence review state;
- question-level results from comparable prior Audits;
- time since the last full-scope inspection;
- source and mapping changes;
- organization, service-provider, aircraft, location, and inspection-type
  comparability; and
- safety-critical and mandatory-control configuration.

Every question receives one visible classification:

- `MANDATORY_CORE`: always included;
- `FOCUSED_FULL`: expanded or full coverage because a risk or change signal is
  present;
- `ROTATIONAL_SAMPLE`: remains in scope with a documented representative
  sample and expansion rule; or
- `DEFER_ELIGIBLE`: may be omitted only after validated clean history,
  unchanged sources, a still-valid full-scope baseline, and explicit approval.

The recommendation must show its signals, history basis, rationale, and
guardrails. The Inspector or Department Manager accepts or changes it with a
recorded reason. The final scope decision is audit logged. Missing records,
unknown history, and “no problem was recorded” cannot be treated as compliance.

Every generated or published question also shows both its
`scopeRecommendation` and `regulatoryTrace`: classification and inclusion or
deferral rationale; source title/version/hash and locator; applicability;
currentness/review state; verification objective; expected Evidence; and exact
origin. A Draft source gap renders the literal `SOURCE_MAPPING_REQUIRED`, not
an empty citation. It cannot support a validation claim, automatic deferral,
publication, or an executable Audit package.

## Steps

1. Open audit
2. Start checklist
3. Answer question
4. Add the required comment and optional mock Evidence filename
5. Create an audit-scoped Potential Finding when eligible
6. Lead Inspector returns, dismisses, or converts the Potential Finding
7. Complete section
8. Submit checklist
9. Generate draft report

## Rules

- Answers: Compliant, Non-Compliant, Observation, Not Applicable, Not Checked
- Non-Compliant and Observation offer `Create Potential Finding` only after a
  required comment is recorded for the exact Audit
- Lead conversion creates the canonical Finding; Inspector execution does not
  issue it directly or switch roles
- Each configured checklist control writes only to its exact Audit; a template
  without an execution package is explicitly disabled
- A submitted checklist stays read-only unless an Inspector or Lead Inspector
  reopens it at a valid stage and records a reason
- A generated mapping or proposed question remains
  `TECHNICAL_REVIEW_REQUIRED` until the responsible Department Manager
  technically approves it within the department scope. Existing
  `EXPERT_REVIEW_REQUIRED` records remain readable legacy compatibility.
- A source gap, including a missing controlled CAA procedure, remains visible
  and prevents the mapping from being treated as validated
- A supplied source version does not by itself make an existing mapping current;
  the immutable source-currentness activation and its impact-review Draft are
  separate from technical approval and publication
- Technical approval of the mapping does not publish a checklist; the
  responsible Department Manager records publication as a separate decision
- One interpreted requirement may produce several practical questions when
  different observations, records, equipment, personnel, or implementation
  controls must be verified
- AI may propose linkages, interpretations, questions, and Evidence
  expectations from controlled sources, but cannot make an official compliance
  finding, legal conclusion, enforcement decision, or publication decision
- Source updates may propose a new mapping or checklist Draft, but cannot
  mutate a published checklist or an in-progress Audit
- A mandatory, safety-critical, newly changed, overdue, open-Finding,
  repeat-Finding, or unknown-history question cannot be automatically omitted
- Any sampled control expands to fuller coverage when an exception is found
- The system enforces a configured full-scope maximum interval even when
  validated prior history is clean

## UX notes

- Show current owner, due date and next action at the top of the screen.
- Keep history in timeline/tab, not as primary content.
- Use primary buttons that match the next action.
- In the Regulatory Library, show the complete source chain, source version,
  review state, source gaps, and why-included rationale.
- In checklist configuration, show compact progressive disclosure for the
  question's scope classification, inclusion/deferral rationale, exact source
  title/version/hash/locator, applicability/currentness/review state,
  verification method, expected Evidence, origin, and reconciliation changes
  without overwhelming inspection execution.
