# Demo MVP Department Manager Decisions

**Decision date:** 20 August 2026
**Status:** Product decision record; implementation not started

## Scope

This record captures the agreed target behavior for the Department Manager
experience. It is not a detailed New Audit screen plan and does not authorize
implementation. New Audit, checklist-item selection, and the remaining
Department Manager panels will each be planned separately before they are
changed.

Checklist-item recommendation and selection remain part of the Department
Manager experience, but their current UI is not changed by these decisions.

## Confirmed decisions

### Terminology

- The creation surface is named `New Audit`.
- The organization field is labelled `Inspected Organization`.
- `Checklist item` is the provisional user-facing term because a checklist may
  currently represent one question/control. The data model must be verified
  before final UI terminology is fixed.

### Scope data

- Inspected Organization, provider scope, regulated target, and inspection type
  remain required system data.
- Provider scope and regulated target are automatically selected when only one
  valid option exists; a dropdown is not shown when the user has no decision to
  make.
- Regulated target uses a human-readable label. Generated fallback text such as
  `... regulated target` is not shown.
- Inspection type remains a user selection because it affects later checklist
  and historical recommendation behavior.

### Trigger and risk

- Trigger type is removed from the New Audit form.
- `Department Manager initiated` is backend-recorded system metadata and may be
  shown read-only in Review or the Inspection Brief.
- The current Risk Category field is removed from Planning because its options
  are inconsistent and its selection has no system effect.
- A future computed risk may be shown read-only with its rationale and sources.
  An authorized override requires a reason.

### Purpose, mode, and location

- Purpose supports both maintained presets and editable free text.
- Inspection approach is not included unless it is later defined as distinct
  from inspection type and mode and has a real downstream effect.
- On-site Audits retain location.
- A location inferred from the regulated target is read-only by default and has
  an adjacent `Edit` action.
- Editing offers previously used locations plus `Enter another location`.
- Existing locations retain their identifier and visible label; manual entries
  are checked for likely duplicates and aliases.
- Remote Audits do not show or require location. An optional online meeting
  link may be supported instead.

### Historical matching

- Location is not a strict historical key.
- Historical comparison primarily uses Inspected Organization, provider scope,
  regulated target, and inspection type.
- Same-location history is a stronger match, but different or unknown location
  does not discard otherwise relevant history.
- Aliases such as `WDH`, `Windhoek`, and `Windhoek International Airport` must
  not create separate historical universes.

### Planning boundary

Department Manager Planning captures:

- selected scope and inspection type;
- purpose, date, and mode;
- location when applicable;
- required inspector count;
- estimated checklist-item count;
- budget information; and
- a reviewable summary for Finance approval.

The system may recommend an estimated checklist-item count and a safe range.
The Department Manager may change the estimate manually. Planning may provide
an optional, explicitly opened filtered preview to help estimate the count, but
it does not select the final checklist items.

Historical recommendation, recommended checklist content, and final
checklist-item selection are removed from the pre-Finance Planning step.

### Post-approval Department Manager work

After the required approval boundary, the approved Audit moves to preparation.
The Department Manager experience continues to own:

- Lead Inspector assignment;
- inspector assignment;
- historical recommendation review;
- recommended checklist review; and
- final checklist-item selection and adjustment.

The detailed checklist-item selection UI will be decided in a separate later
design pass and must not be changed during the first Planning implementation.

### Planning home

- Past plans, upcoming plans, in-progress Audits, plans awaiting approval, and
  approved plans must be easy to find.
- Plans must be sortable by date and filterable by relevant workflow and scope
  attributes.
- Time groupings and approval/workflow statuses must remain distinguishable in
  the UI.
- Approved Audits requiring team/checklist preparation must lead to the
  Department Manager preparation area.

## Decisions deliberately left open

- Whether automatically selected provider scope and regulated target are shown
  in the main flow or only in Review.
- The detailed New Audit information architecture and visual design.
- The exact relationship and final labels for checklist, checklist item,
  question, and control.
- Purpose preset content.
- Whether Hybrid is a supported mode.
- Online meeting-link storage and behavior.
- Detailed historical recommendation and checklist-item selection UI.
- Final cross-role approval and release sequence for the simplified demo.
- Final whole-system UI design after all role workflows are agreed.

## Current outcome

Decisions are recorded. Source, API, data model, fixture, and UI implementation
are unchanged. Implementation and verification are `not run`.
