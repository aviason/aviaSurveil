# Surveillance Planning Workflow

## Purpose

Create annual, routine, and ad hoc audit plans without a complex scheduling
tool, while applying the configured advance-notice policy before execution.

## Steps

1. Department Manager prepares and submits annual planning
2. Finance reviews budget and resources
3. General Manager reviews and forwards the plan
4. Executive Director approves the plan with a demo-only mock mark
5. General Manager performs the separate `GM Release to Department` step
6. Department prepares the released plan
7. Select period/year, organization, active provider scope, typed regulated target,
   audit type, domain, location, and planned date
8. Department Manager assigns the Lead Inspector
9. Lead Inspector assigns the inspection team and checklist template
10. Evaluate the configured advance-notice policy
11. For Routine / Announced inspections, send the proposed date, checklist,
   and relevant information to the Service Provider
12. Let the Service Provider confirm the proposed date or provide an
    alternative date; the CAA confirms any alternative
13. For Ad Hoc / Unannounced inspections, skip advance notification and keep
    the coordination package unavailable to the Service Provider
14. Mark the inspection team and schedule ready for execution
15. Publish to calendar

## Rules

- Manual scheduling in MVP
- Planning approval order is Department Manager -> Finance -> General Manager
  -> Executive Director -> GM Release to Department -> Department preparation.
- Executive Director approval leaves preparation at `not_released`; it does
  not absorb or bypass the General Manager release step.
- Risk score informational only in MVP
- Reschedule requires reason
- Published audit appears on inspector dashboard
- An organization may retain multiple active provider scopes. A scope may bind
  an organization, person, facility, device, system, asset, or location and is
  not inferred from the coarse organization type.
- The responsible Department Manager technically approves applicable candidate
  content and separately records publication before it can be selected.
- The inspection type/configured policy determines whether advance notice is
  required; do not infer this from UI color or free text.
- Routine / Announced execution becomes ready only after the proposed date is
  confirmed or a Service Provider alternative is accepted by the CAA.
- Ad Hoc / Unannounced inspections do not create an advance Service Provider
  notification, portal request, or shared checklist package.
- Demo notifications and date responses remain browser-local; no real email,
  calendar invitation, or external delivery is claimed.

## UX notes

- Show current owner, due date and next action at the top of the screen.
- Keep history in timeline/tab, not as primary content.
- Use primary buttons that match the next action.
