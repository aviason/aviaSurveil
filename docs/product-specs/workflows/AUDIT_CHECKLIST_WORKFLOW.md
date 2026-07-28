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
9. Have the relevant technical expert validate the source chain, applicability,
   interpretation, questions, and Evidence.
10. Let the authorized publication owner incorporate validated questions into a
    versioned checklist.

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
6. Present the impact set to the relevant technical expert. Download or
   extraction success does not satisfy this validation gate.
7. Put accepted changes into a new checklist Draft. Keep published checklist
   versions and in-progress Audits immutable.
8. Perform a complete expert validation at least annually even if the
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
  `EXPERT_REVIEW_REQUIRED` until the relevant technical expert validates it
- A source gap, including a missing controlled CAA procedure, remains visible
  and prevents the mapping from being treated as validated
- Technical validation of the mapping does not publish a checklist; the
  Department Manager remains the publication owner
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
  question's mapping identity, verification method, Evidence expectation, and
  rationale without overwhelming inspection execution.
