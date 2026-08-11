# Status, Permission, Security and Audit Rules

## Audit statuses

Draft, Planned, Scheduled, In Progress, Checklist Completed, Draft Report, Report Issued, Follow-up Open, Closed, Cancelled.

## Finding statuses

Draft, Open, Waiting for CAP, CAP Submitted, CAP Accepted, CAP Rejected, Evidence Required, Evidence Submitted, Pending CAA Review, More Information Requested, Overdue, Pending Closure, Closed, Escalated.

## CAP statuses

Draft, Submitted, Pending CAA Review, Accepted, Rejected, More Information Requested, Superseded.

## Evidence statuses

Uploaded, Pending CAA Review, Accepted, Partially Accepted, Rejected, More Information Requested, Superseded.

## Permissions

| Action | Inspector | Lead Inspector | Department Manager | Finance | General Manager | Executive Director | Auditee | Admin |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| View assigned audits | Yes | Yes | Yes | No | Summary | Summary | Own organization | Yes |
| Coordinate announced inspection date | No | Send / confirm alternative | View | No | Summary | Summary | Confirm / propose alternative for own organization | No |
| Create audit | Limited | Yes | Yes | No | No | No | No | Yes |
| Complete checklist | Yes, assigned scope | Yes | No | No | No | No | No | No |
| Create / issue Finding | Create | Review / issue | Yes | No | No | No | No | No |
| Submit CAP / Evidence | No | No | No | No | No | No | Own organization | No |
| Review CAP | Yes | Yes | Summary | No | Summary | Summary | No | No |
| Record Evidence verification: Close / Partially Close / Not Close | Yes | Yes | No | No | No | No | No | No |
| Close Finding | Verification `Close` | Verification `Close` | Separate authorized path | No | No | No | No | No |
| Review budget-required plan | No | No | Submit | Approve / return only | Review / release stage | Final plan approval (demo) | No | No |
| Review Preliminary Report | No | Prepare | Department review | No | Return / forward only | Final decision | ED-released own-organization only | No |
| Issue, mock-sign, or lock Preliminary Report | No | No | No | No | No | Yes, demo-only | No | No |
| Review Final Report | No | Prepare | Department review | No | Return / forward only | Final decision | Issued own-organization only | No |
| Issue, mock-sign, or lock Final Report | No | No | No | No | No | Yes, demo-only | No | No |
| Configure templates | No | Limited | Limited | No | No | No | No | Yes |
| Start or import a checklist generation run | No | No | Yes, own scope | No | No | No | No | Yes, candidate only |
| Activate an immutable source-currentness transition | No | No | No | No | No | No | No | Yes, candidate only; not technical approval or publication |
| Technically approve a checklist candidate after technical review | No | No | Yes, current responsible department assignment and own scope | No | No | No | No | No |
| Publish a technically approved checklist version | No | No | Yes, current responsible department assignment and own scope; separate recorded action | No | No | No | No | No |

## Security rules

- Auditee can see only its own organization across Messages, Settings, counts, documents, reports, CAP, Evidence, and notifications.
- Auditee can see advance coordination only for its own Routine / Announced inspections after the CAA releases the request. Ad Hoc / Unannounced and notice-withheld records never render in advance.
- Auditee cannot see Internal CAA Notes, inspector workload, other organizations, internal risk scoring, private dashboards, or enforcement deliberations.
- Department Manager forwards Preliminary and Final Reports to General Manager;
  Department Manager cannot issue, share, mock-sign, or lock them.
- General Manager review is intermediate. GM can return or forward a
  Preliminary or Final Report but cannot issue, share, mock-sign, or lock it.
- Executive Director alone may issue, mock-sign, and lock an eligible
  Preliminary or Final Report in this demo.
- Service Provider visibility for a Preliminary Report begins only after
  Executive Director release and remains restricted to the report's
  `organizationId`. `capRequired` changes the recipient action, not the approval
  chain.
- Finance can approve or return a budget-required plan only; Finance cannot edit audit scope, release a plan, or decide a report.
- Planning approval order: Department Manager -> Finance Review -> GM Review -> Executive Director.
  Finance approval advances to GM Review. Finance Return for Revision goes to Department Manager.
  GM may forward a Finance-reviewed plan to Executive Director or return it to Department Manager.
  A corrected submission must pass Finance again. Executive Director approval does not release the plan directly; GM Release to Department remains a separate next action.
- CAP acceptance is not Finding closure. Report approval does not close open
  Findings or bypass required Evidence/verification. Only Inspector or Lead
  Inspector records `Close`, `Partially Close`, or `Not Close` in this demo.
  `Partially Close` and `Not Close` leave the Finding at
  `EVIDENCE_MORE_INFO`; only `Close` changes it to `CLOSED`.
- `Comment to Auditee` and `Internal CAA Note` are both required for CAP
  verification and remain separate. Internal notes never render to the Service
  Provider.
- Enforcement remains a separate configured or authorized process and is never automatic.
- Submitted evidence is never deleted.
- Demo traceability uses browser-local audit history; it is not a production
  immutable audit trail.
- Internal users should use MFA.
- Sensitive data and files need access control.
- A generic `manager` role without a current responsible department assignment
  has no technical-review or publication authority. Admin and generation
  providers may create or edit candidates but cannot technically approve or
  publish them.
- Source-currentness activation is append-only and records an exact
  predecessor/current source transition. It may create an impact-review Draft,
  but never changes a published checklist or in-progress Audit and is not a
  legal interpretation, Department Manager technical approval, or publication
  decision.
- `TECHNICAL_REVIEW_REQUIRED` is the new workflow status. Existing
  `EXPERT_REVIEW_REQUIRED` records remain readable as legacy compatibility and
  are never treated as approved merely by being read.

## Governed AGA intake authority and privacy boundary

The eight top-level roles remain Inspector, Lead Inspector, Department Manager,
Finance, General Manager, Executive Director, Auditee, and Admin. Functional
assignments are not top-level roles: `REGULATORY_SOURCE_OWNER` may attest only
the exact assigned source scope, and `CHECKLIST_REVIEWER` may comment or make a
non-binding recommendation only. A source-owner or reviewer assignment never
confers Admin, author, technical-approval, publication, or Audit-package power.

Client Department/unit input is only an untrusted selection hint. The server
derives required owners; missing or ambiguous ownership fails closed. Admin may
receive inventory and prepare candidate Drafts but cannot establish source
authority, technically approve, publish, or evaluate an executable package as
eligible. An Auditee cannot view intake inventory, raw extraction, source-owner
or reviewer comments, Internal CAA Note, required-owner facts, technical
decisions, publication deliberation, or other organizations' records.

`SOURCE_MAPPING_REQUIRED` is a literal source-gap state, not an empty trace.
Source-authority acceptance, candidate mapping attestation, technical approval,
publication, and package eligibility are distinct facts. Publication does not
imply Audit-package eligibility, and no role bypass is permitted.

Implementation evidence is **demo-only** and **verified locally** through focused Node regressions and fresh isolated-browser authority/privacy checks at `1536x864`, `1366x768`, `1024x768`, and `390x844`. Production identity, authorization, signing, enforcement execution, and immutable audit logging are **not run**; **production-readiness not claimed**.

Approvals and timestamps are browser-local mock records. Attachments are mock
filenames in local browser state; no secure document storage is implemented.

## Retired AGA candidate donor privacy boundary

The separate `aga-candidate-demo@1.1.0` capability, tagged reader, writer,
schema, and panel were physically removed under canonical Task 9 after the
user's explicit `delete` decision. No current role or credential can access
that donor. Historical privacy receipts remain candidate-only evidence; the
canonical exercise catalog continues to enforce scope authority, private
storage, and browser/telemetry residue denial.
