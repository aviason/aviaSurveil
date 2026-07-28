# Part 127 / Part 140 Candidate Applicability Assessment

This note is the human-readable companion to
[`ncaa-namcats-part-127-140-applicability.json`](ncaa-namcats-part-127-140-applicability.json).
The JSON record is the machine-readable authority for source identities,
hashes, page and section locators, checklist implications, and approval gates.

## Status and scope

- Assessment: `CANDIDATE_DERIVED_CONTEXT`
- Evidence: `SOURCE_BOUND`
- Review: `EXPERT_REVIEW_REQUIRED`
- Publication: `NOT_PUBLISHED_BY_THIS_ASSESSMENT`
- Production readiness: `NOT_CLAIMED`
- Pilot: OPS / Air Operator (AOC), PQ 4.450, mapping
  `RMAP-OPS-AOC-CABIN-RAMP-001`

This assessment records what the downloaded public sources support. It is not
legal advice, an official NCAA interpretation, or authority to publish a
checklist.

## Source-text retention

The downloaded PDFs and complete extracted text remain in the ignored local
vault under:

```text
.local/aviasurveil360/regulatory-sources/ncaa/namcats/all-pages/
```

The tracked JSON record points to each full-text file and binds it to the exact
source URL, byte size, SHA-256 digest, extraction result, page count, and
manifest identity. It stores paraphrased evidence and exact page/section
locators rather than duplicating the full regulatory text in Git.

## Candidate conclusions

### Part 127

Classification: `OPERATION_TYPE_CONDITIONAL`.

Part 127 is the technical-standard source for commercial helicopter
operations. It is relevant only when the inspected AOC, aircraft, and activity
fall within that operation type and any narrower condition in the cited
clause. It must not be applied automatically to fixed-wing Part 121 or Part 135
operations.

The reviewed text provides candidate support for:

- flight-crew seats, passenger and crew restraints, cabin and galley securing,
  passenger seating and evacuation, briefing, and generic emergency-equipment
  checks in section 127.04.2 on PDF page 76;
- oversight finding classification, corrective-action handling, and records in
  section 127.06.5 on PDF pages 115-117;
- conditional refuelling/egress controls on PDF page 120;
- a narrow passenger-seat/restraint requirement for the stated single-engine
  night operation on PDF page 133;
- MEL, technical-log, secure-stowage, emergency-equipment access, passenger
  briefing, and safety-card content on PDF pages 138-143.

No direct extracted-text match was found for `lavatory`, `protective breathing
equipment`, or `PBE`. This is a visible source gap, not proof that no applicable
requirement exists in another regulation, technical standard, approved
operator manual, aircraft configuration, maintenance source, or controlled
NCAA procedure.

### Part 140

Classification: `SYSTEM_LEVEL_APPLICABLE`.

The 2025 Part 140 text expressly includes an air operator and requires NCAA
acceptance and surveillance of the SMS through document review and auditing.
It supports organization-level verification of SMS documentation, hazard and
risk management, safety assurance, performance monitoring, management of
change, internal/external audits, corrective-action monitoring, and records.

Part 140 is therefore relevant to checklist-level risk and assurance context.
It is not the sole direct regulatory basis for individual galley, lavatory,
seat, PBE, display, or emergency-exit serviceability questions.

The NCAA library also lists a signed 2021 Part 140 file. The 2025 Revision 2
file is newer on its face, but simultaneous public listing does not prove
formal supersession. The JSON record keeps the 2021 file as a comparator and
requires a source owner to confirm current authority.

## Six-question impact summary

| Question | Part 127 | Part 140 | Required next decision |
|---|---|---|---|
| `CAB-GALLEY-001` | Conditional direct basis for applicable helicopter galley/stowage configuration | Risk and assurance context only | Confirm operation, configuration, manual/MEL, and national-clause linkage |
| `CAB-LAV-001` | No direct match in the reviewed source | Risk and assurance context only | Identify a direct lavatory/configuration/maintenance/procedure source |
| `CAB-PAX-SEAT-001` | Conditional direct basis; PDF page 133 is limited to the stated operation | Risk and assurance context only | Confirm general versus condition-specific seat/restraint applicability |
| `CAB-EMEQ-PBE-001` | Generic emergency-equipment context; no PBE-specific match | Risk and assurance context only | Identify the PBE-specific national, approved-manual, configuration, maintenance, and inspection sources |
| `CAB-VID-CREW-SEAT-001` | Conditional partial basis for crew seat/restraint, not the combined display element | Risk and assurance context only | Split or separately source the display and crew-seat requirements |
| `CAB-COCKPIT-GEN-001` | Conditional partial basis for exit, marking, stowage, and evacuation elements | Risk and assurance context only | Decompose the broad prompt or add direct sources for each retained objective |

## Human gates

Before any mapping can become `VALIDATED`:

1. An NCAA source owner must confirm the current authoritative versions.
2. An Operations technical expert must confirm operator, aircraft,
   operation-type, configuration, and clause applicability.
3. The controlled NCAA Operations surveillance/ramp-inspection procedure must
   be supplied or identified.
4. A technical expert must validate question decomposition, verification
   methods, and expected Evidence.
5. A Department Manager or configured publication owner must separately
   approve a new checklist version.

This assessment cannot automatically decide compliance, severity, enforcement,
certification, CAP acceptance, Evidence acceptance, or Finding closure.
