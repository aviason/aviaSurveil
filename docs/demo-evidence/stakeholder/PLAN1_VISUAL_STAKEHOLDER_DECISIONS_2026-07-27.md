# Plan 1 Visual Stakeholder Decisions

**Decision date started:** 27 July 2026

**Plan:** [Full React 86-Screen Migration](../../exec-plans/completed/2026-07-22-full-react-86-screen-migration-plan.md)

**Source ledger:** [React 86-Screen Visual Review Ledger](../REACT_86_SCREEN_VISUAL_REVIEW_2026-07-25.md)

**Scope:** The 170 route/viewport pairs that retained decoded-pixel failures in
the final exact-runtime visual command.

**Disposition progress:** 170/170 resolved

**Manual review remaining:** 0/170

On 28 July 2026, the user directed the Plan 1 closure to apply Codex's
high-confidence recommendations and leave only the records explicitly marked
as requiring manual stakeholder review. This authorizes an aggregate Accept
disposition for 160 records whose triage has
`codexRecommendation=accept` and `userReviewRequired=false`.

The user then directed the remaining nine risk-surface records to be fixed:
make the Manager Risk Dashboard heatmap more visible, show the Organization
Risk Profile health score, and preserve the GM Risk Dashboard risk matrix.
All nine implementations are `fixed-verified-locally`. The focused visual
command remains literally non-green for the nine route/viewport pixel
comparisons; no baseline, threshold, mask, authority, or semantic truth was
weakened.

## Decision Records

Sequence | Audit ID | Surface | Viewport | Failed region and ratio | User decision | Brief rationale | Decision date | Implementation status
---|---|---|---|---|---|---|---|---
1 | ui-audit-003 | `inspector-findings` | desktop | content 0.10423/0.08000 | Fix | The correction restored the accepted nine-record queue total, source-faithful desktop queue composition, and styled dossier navigation while preserving canonical React Finding identity. | 2026-07-27 | fixed-verified-locally
2 | ui-audit-003 | `inspector-findings` | tablet | content 0.08499/0.08000 | Accept | Stakeholder accepted the tablet pixel deviation despite shifted candidate content and action presentation. | 2026-07-27 | accepted
3 | ui-audit-005 | `inspector-calendar` | tablet | content-header 0.10239/0.08000 | Accept | Stakeholder accepted this visual deviation for Plan 1 sign-off. | 2026-07-27 | accepted
4 | ui-audit-005 | `inspector-calendar` | mobile | content 0.10156/0.08000 | Accept | Stakeholder accepted this visual deviation for Plan 1 sign-off. | 2026-07-27 | accepted
5 | ui-audit-006 | `inspector-reports` | mobile | content 0.08903/0.08000 | Accept | Stakeholder accepted this visual deviation for Plan 1 sign-off. | 2026-07-27 | accepted
6 | ui-audit-007 | `audit-detail` | desktop | content 0.08752/0.08000 | Accept | Stakeholder accepted this visual deviation for Plan 1 sign-off. | 2026-07-27 | accepted
7 | ui-audit-007 | `audit-detail` | tablet | content 0.09592/0.08000 | Accept | Stakeholder accepted this visual deviation for Plan 1 sign-off. | 2026-07-27 | accepted
8 | ui-audit-008 | `checklist-runner` | desktop | content 0.11557/0.08000 | Accept | Stakeholder accepted this visual deviation for Plan 1 sign-off. | 2026-07-27 | accepted
9 | ui-audit-008 | `checklist-runner` | tablet | content 0.13639/0.08000 | Accept | Stakeholder accepted this visual deviation for Plan 1 sign-off. | 2026-07-27 | accepted
51 | ui-audit-031 | `manager-risk-dashboard` | desktop | content 0.12538/0.08000 | Fix | Stakeholder required the risk heatmap to be more visible. The candidate now places an accessible 25-cell Risk Exposure Matrix directly after the filters and before the indicator cards. | 2026-07-28 | fixed-verified-locally
52 | ui-audit-031 | `manager-risk-dashboard` | tablet | content 0.08693/0.08000 | Fix | Stakeholder required the risk heatmap to be more visible. The candidate now places an accessible 25-cell Risk Exposure Matrix directly after the filters and before the indicator cards. | 2026-07-28 | fixed-verified-locally
53 | ui-audit-031 | `manager-risk-dashboard` | mobile | content 0.13349/0.08000 | Fix | Stakeholder required the risk heatmap to be more visible. The candidate now places an accessible 25-cell Risk Exposure Matrix directly after the filters and before the indicator cards. | 2026-07-28 | fixed-verified-locally
63 | ui-audit-037 | `organization-risk-profile` | desktop | sidebar 0.04850/0.03000; content 0.08219/0.08000 | Fix | Stakeholder required the organization risk score to be shown. The configured Fly Namibia demo scenario now displays Oversight Health 74, Needs Attention, its configured-demo basis, and the recommended action. | 2026-07-28 | fixed-verified-locally
64 | ui-audit-037 | `organization-risk-profile` | tablet | sidebar 0.05928/0.03000; content 0.12610/0.08000 | Fix | Stakeholder required the organization risk score to be shown. The configured Fly Namibia demo scenario now displays Oversight Health 74, Needs Attention, its configured-demo basis, and the recommended action. | 2026-07-28 | fixed-verified-locally
65 | ui-audit-037 | `organization-risk-profile` | mobile | content 0.19173/0.08000 | Fix | Stakeholder required the organization risk score to be shown. The configured Fly Namibia demo scenario now displays Oversight Health 74, Needs Attention, its configured-demo basis, and the recommended action. | 2026-07-28 | fixed-verified-locally
113 | ui-audit-056 | `gm-risk-dashboard` | desktop | content 0.15107/0.08000 | Fix | Stakeholder required the GM risk matrix heatmap to be preserved. The candidate now renders an accessible 25-cell Risk Exposure Matrix immediately after the GM risk KPIs. | 2026-07-28 | fixed-verified-locally
114 | ui-audit-056 | `gm-risk-dashboard` | tablet | content 0.22749/0.08000 | Fix | Stakeholder required the GM risk matrix heatmap to be preserved. The candidate now renders an accessible 25-cell Risk Exposure Matrix immediately after the GM risk KPIs. | 2026-07-28 | fixed-verified-locally
115 | ui-audit-056 | `gm-risk-dashboard` | mobile | content 0.14965/0.08000 | Fix | Stakeholder required the GM risk matrix heatmap to be preserved. The candidate now renders an accessible 25-cell Risk Exposure Matrix immediately after the GM risk KPIs. | 2026-07-28 | fixed-verified-locally

## Manual Decisions Remaining

Sequence | Audit ID | Surface | Viewport | Failed region and ratio | Decision question
---|---|---|---|---|---
