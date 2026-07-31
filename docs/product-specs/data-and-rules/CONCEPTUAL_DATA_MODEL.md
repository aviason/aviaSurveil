# Conceptual Data Model

## Core entities

- User
- Role
- Organization
- OrganizationUser
- Audit
- AuditTeamMember
- ChecklistTemplate
- ChecklistTemplateVersion
- ChecklistQuestion
- ChecklistResponse
- RegulatorySourceVersion
- RegulatorySourceCurrentnessActivation
- SourceImpactReviewDraft
- GenerationRunSourceCurrentnessBinding
- RegulatoryReferenceSnapshot
- RegulatoryMapping
- RegulatoryRequirement
- ProposedChecklistQuestion
- RegulatorySourceCollection
- ServiceProviderCatalog
- ServiceProviderScope
- RegulatedTarget
- DepartmentReviewDecision
- ChecklistPublicationDecision
- RegulatoryRefreshRun
- DerivedRegulatoryAssessment
- ChecklistScopeRecommendation
- Finding
- CAP
- Evidence
- ReviewDecision
- Notification
- Report
- AuditLog

## Relationships

Organization has many Audits.
Audit has many ChecklistResponses.
Audit has many Findings.
Finding has many CAP revisions.
Finding has many Evidence versions.
CAP may have Evidence.
Evidence has ReviewDecisions.
Finding has status history and audit logs.
RegulatoryReferenceSnapshot has many RegulatoryMappings.
RegulatoryMapping references exact RegulatorySourceVersions.
RegulatoryMapping has one interpreted RegulatoryRequirement and one
verification objective.
RegulatoryRequirement may have many ProposedChecklistQuestions.
RegulatorySourceCollection has many immutable source-file identities and
RegulatoryRefreshRuns.
RegulatorySourceCurrentnessActivation binds one exact current
RegulatorySourceVersion/hash to its optional exact predecessor for one source
identity. A changed activation has one SourceImpactReviewDraft; that Draft may
link several independently imported candidate roots without changing any
published checklist or Audit.
GenerationRunSourceCurrentnessBinding proves that a generated candidate bound
to a previously committed activation; candidate import cannot create an
activation implicitly.
Organization has one coarse organization type for compatibility and may have
several provider scopes simultaneously. A ServiceProviderScope is an
approval/oversight scope, not a replacement legal organization type, and it
may target an ORGANIZATION, PERSON, FACILITY, DEVICE, SYSTEM, ASSET, and LOCATION.
ServiceProviderCatalog preserves the exact provider label, topics, raw
responsible CAA unit, aliases, target kinds, and normalization state for each
versioned provider scope.
DerivedRegulatoryAssessment references exact RegulatorySourceVersions and has
many page-bound EvidenceRecords, ApplicabilityDecisions, and
ChecklistImplications.
ChecklistScopeRecommendation references an exact Audit, checklist version,
question set, source snapshot, and history-signal snapshot.
An approved ProposedChecklistQuestion may resolve to an exact
ChecklistQuestion identity; ChecklistResponses retain that exact question and
template-version identity rather than copying the entire source graph.

## Critical modeling rules

1. Do not merge CAP and Evidence.
2. Do not delete submitted evidence; version it.
3. Separate internal CAA note from auditee-visible comment.
4. Every status transition must be audit logged.
5. Every finding must have current owner and next action.
6. Overdue can be computed from due date, but visible as label.
7. Keep source title, version or date, locator, source type, and review status
   with every regulatory mapping. A public download index is not a substitute
   for an exact promulgated or controlled source version.
8. Keep PQ, Critical Element, Annex/SARP, national regulation, CAA
   procedure/guidance, interpreted requirement, verification objective,
   practical question, and expected Evidence as separate fields. Do not flatten
   the chain into one unreviewable citation string.
9. One requirement may generate several practical inspection questions. Do not
   assume one regulation clause equals one checklist question.
10. New generated mappings and questions start as `TECHNICAL_REVIEW_REQUIRED`.
    The currently assigned responsible Department Manager confirms source
    applicability, interpretation, question decomposition, and Evidence
    expectation within the manager's department scope. Existing
    `EXPERT_REVIEW_REQUIRED` records remain readable as explicit legacy
    compatibility only; they are not relabelled as approved.
11. A missing controlled CAA procedure, uncertain service-provider
    applicability, or unverified crosswalk stays visible as a source gap. The
    system must not silently promote that mapping to `VALIDATED`.
12. Technical approval and checklist publication are separate Department
    Manager decisions with separate audit events and timestamps. Department
    Manager publication authority does not make the underlying regulatory
    interpretation legally authoritative.
13. Regulatory material is a configured reference or Finding basis, not legal
    advice or an automatic compliance, enforcement, certification, or closure
    decision.
14. Download, text extraction, clause mapping, technical review, and checklist
    publication are distinct states. A successful download or extraction must
    never be displayed as a validated regulatory interpretation.
15. A source change creates an append-only impact-review proposal. Existing
    Audits stay pinned to their exact checklist version; accepted changes enter
    a new Draft and require the normal technical-validation and publication
    gates.
16. Checklist scope recommendations are advisory snapshots. They must expose
    the exact signals, history basis, classification, rationale, approver, and
    resulting scope decision.
17. Absence of a recorded problem is not compliance evidence. Unknown or
    insufficient history cannot make a question deferrable.
18. Mandatory, safety-critical, newly changed, overdue, open-Finding,
    repeat-Finding, and source-gap controls cannot be automatically omitted.
    Even validated clean history retains a configured full-scope maximum
    interval.
19. A derived regulatory assessment stores conclusions and evidence locators,
    not copied full regulatory text. Every conclusion remains bound to an exact
    source identity and hash; a hash change invalidates the affected
    source-bound conclusion and creates an impact-review proposal.
20. Applicability is typed. `OPERATION_TYPE_CONDITIONAL`,
    `SYSTEM_LEVEL_APPLICABLE`, direct, partial, contextual, and no-direct-match
    dispositions must not be flattened into a generic “applicable” flag.
21. A negative extracted-text search is a source-gap signal, not proof that no
    requirement exists. Technical review must consider other regulations,
    technical standards, approved manuals, configurations, maintenance
    sources, and controlled CAA procedures.
22. Observing or downloading a source version does not establish currentness.
    Currentness is an explicit append-only activation with exact
    predecessor/current snapshot hashes; a historical version cannot be
    reactivated after a newer head. Activation creates a review boundary, not a
    technical, legal, or publication approval.
23. Every generated or published ProposedChecklistQuestion contains both a
    `scopeRecommendation` (classification, signals, operational-history basis,
    rationale, guardrails, approval/review state) and a `regulatoryTrace`
    (source identity/title, immutable version/SHA-256, locator/page/section/
    clause, source type, applicability, national/CAA mapping, verification
    objective, expected Evidence, currentness, and technical-review state).
24. An explicit `SOURCE_MAPPING_REQUIRED` trace is allowed only as a
    repairable candidate Draft and never as an empty or partial citation. It
    blocks validation claims, automatic deferral, executable Audit-package
    materialization, technical approval, and publication.
25. Each candidate question records exactly one origin:
    `REGULATORY_TRACE`, `EXISTING_CHECKLIST_CANDIDATE`, or
    `HYBRID_RECONCILED`. Existing wording, operational intent, and result
    history remain non-authoritative candidate input; hybrid reconciliation
    makes wording, Evidence, applicability, and scope differences reviewable
    against the current regulatory/controlled-CAA-procedure chain.

## Regulatory knowledge pilot fields

### RegulatorySourceVersion

- id
- title
- source_type
- version_or_date
- status (`SUPPLIED_WORKING_COPY`, `PUBLIC_REFERENCE`, or `SOURCE_GAP`)
- locator
- public_url, when applicable

### RegulatoryMapping

- id
- audit_area
- service_provider_types
- applicable_regulation_scope
- critical_element
- protocol_question_id
- protocol_question
- annex_sarp_references
- national_regulation_references
- caa_implementation_reference
- requirement
- verification_objective
- expected_evidence
- why_included
- review_status (`TECHNICAL_REVIEW_REQUIRED`, `VALIDATED`, or `REJECTED`;
  existing `EXPERT_REVIEW_REQUIRED` is readable legacy compatibility)
- source_gap
- source_version_ids

### ProposedChecklistQuestion

- id
- regulatory_mapping_id
- prompt
- verification_method
- evidence_examples
- why_included
- review_status inherited from or more restrictive than its mapping

### RegulatorySourceCollection

- id
- source_page_url
- source_scope
- synchronized_at
- source_file_count
- source_file_url
- source_file_name
- observed_updated_at
- observed_size
- byte_size
- sha256
- extraction_status
- extracted_text_locator
- event_driven_review
- reconciliation_interval_months
- technical_validation_interval_months
- source_change_state
- tracked_manifest_locator

The initial candidate collection is bounded to all three public NCAA NAMCATS
library pages. It stores each unique PDF once, records every page on which a
source is listed, and includes the linked index workbook. Source bytes,
extracted text, and resumable page-level OCR checkpoints live in an ignored
local vault. Searchable pages retain their embedded text; individual
image-only pages are processed with local OCR and merged in document order.
The tracked manifest records identity, hashes, listing pages, and extraction
state without placing full regulatory content in Git.

### DerivedRegulatoryAssessment

- id
- assessment_scope
- assessment_status (`CANDIDATE_DERIVED_CONTEXT`)
- evidence_status (`SOURCE_BOUND`)
- review_status (`TECHNICAL_REVIEW_REQUIRED`, `VALIDATED`, or `REJECTED`;
  existing `EXPERT_REVIEW_REQUIRED` is readable legacy compatibility)
- publication_status
- source_version_id
- source_sha256
- extracted_text_locator
- evidence_id
- pdf_pages
- section
- paraphrased_evidence
- candidate_use
- limitation
- applicability_classification
- checklist_question_id
- source_disposition
- candidate_conclusion
- required_department_manager_action
- governance_gates
- guardrails

The current candidate Part 127 / Part 140 assessment lives under
`docs/regulatory-sources/derived/`. Part 127 is
`OPERATION_TYPE_CONDITIONAL`: it may apply to a commercial-helicopter operation
only after the exact AOC, aircraft, operation, configuration, and clause are
confirmed. Part 140 is `SYSTEM_LEVEL_APPLICABLE` to the air operator's SMS,
risk, audit, assurance, change, corrective-action-monitoring, and records
controls; it is not the sole direct basis for each cabin-equipment question.
Current-source authority, controlled procedure, interpretation, question
decomposition, expected Evidence, and checklist publication remain separate
human gates.

### ChecklistScopeRecommendation

- id
- audit_id
- checklist_template_version_id
- source_snapshot_id
- generated_at
- status (`ADVISORY_ONLY`)
- history_state (`INSUFFICIENT_FOR_DEFERRAL` or
  `VALIDATED_HISTORY_AVAILABLE`)
- question_id
- classification (`MANDATORY_CORE`, `FOCUSED_FULL`, `ROTATIONAL_SAMPLE`, or
  `DEFER_ELIGIBLE`)
- signals
- history_basis
- rationale
- guardrails
- requires_manager_approval
- accepted_scope
- decision_rationale
- decided_by
- decided_at

`DEFER_ELIGIBLE` is not an automatic omission. It is available only after the
configured clean-history, unchanged-source, baseline/full-scope, and approval
conditions have been met and recorded.

## Minimal Finding fields

- id
- finding_number
- audit_id
- organization_id
- title
- description
- regulation_reference
- severity
- due_date
- status
- current_owner_type
- current_owner_id
- next_action
- cap_required
- evidence_required
- repeat_finding_flag
- created_by
- issued_at
- closed_at
