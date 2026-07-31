package testprofile

const opsAOCCabinRampRegulatorySnapshot = `{
  "versionLabel": "2026.1",
  "configuredRules": [
    "Configured reference for Cabin Inspection sampling",
    "Candidate-only trace from OPS PQ 4.450 / CE-7 to practical cabin/ramp verification",
    "AI draft cannot publish itself; responsible Department Manager technical review is required"
  ],
  "changeHistory": [
    "2026-01-01 - Added to the mock regulatory library",
    "2026-07-28 - Added supplied-source OPS / Air Operator trace pilot"
  ],
  "mappings": [
    {
      "id": "RMAP-OPS-AOC-CABIN-RAMP-001",
      "auditArea": "OPS",
      "serviceProviderTypes": ["Air Operator (AOC)"],
      "applicableRegulations": ["Part 91", "Part 92", "Part 121 / 127 / 135", "Part 140"],
      "criticalElement": "CE-7",
      "protocolQuestionId": "4.450",
      "protocolQuestion": "Does the surveillance programme include risk-based ramp inspections of AOC holders and foreign air operators?",
      "annexReferences": [
        "Annex 6 Part I 4.2.2.2",
        "Annex 6 Part I Appendix 5 section 7",
        "Annex 6 Part I 4.2.12.1",
        "Annex 6 Part I 6.2.2",
        "Annex 6 Part I 6.16.1-6.16.3",
        "Annex 6 Part I 8.1.1"
      ],
      "nationalReferences": [
        "NAMCAR 121.07.6-121.07.8",
        "NAMCAR 135.07.6-135.07.8",
        "NAMCAR 91.04.14, 91.04.16-91.04.17, 91.04.21-91.04.22, 91.04.24, 91.04.27-91.04.29",
        "NAMCAR 121.02.14, 121.05.21, 121.05.24-121.05.26, 121.05.30-121.05.38, 121.05.45, 121.05.48",
        "NAMCAR 135.05.17-135.05.18, 135.05.22, 135.05.25-135.05.26, 135.05.32-135.05.33",
        "NAMCAR 91.10.1, Part 121 Subpart 10, Part 135 Subpart 10, and Part 145"
      ],
      "caaImplementationReference": "NCAA Operations surveillance / ramp-inspection procedure - controlled source not supplied",
      "requirement": "The NCAA establishes and executes a risk-based ramp-surveillance programme that uses standardized inspection coverage, preserves inspection records, and follows identified discrepancies through the authorized safety-issue process.",
      "verificationObjective": "During a scoped cabin/ramp inspection, verify that selected cabin safety equipment, restraints, seats, exits, placards, and associated control records are present, serviceable, accessible, and consistent with the operator's approved documentation.",
      "expectedEvidence": [
        "Inspector observation recorded against the exact aircraft and inspection",
        "Approved operations manual, MEL, or controlled equipment reference",
        "Serviceability, maintenance, expiry, or defect-control record as applicable",
        "Permitted photo or attachment linked to the exact checklist response",
        "Inspection discrepancy and follow-up record when an exception is identified"
      ],
      "whyIncluded": "OPS PQ 4.450 identifies risk-based ramp inspection and cabin/safety coverage. The supplied Annex 6 Part I crosswalk links the surveillance obligation and selected cabin/equipment SARPs to NAMCAR clauses. The six practical questions are a candidate decomposition for responsible Department Manager technical review, not an official compliance conclusion.",
      "reviewStatus": "EXPERT_REVIEW_REQUIRED",
      "sourceGap": "The controlled NCAA Operations surveillance/ramp-inspection procedure has not been identified or supplied. Applicability and the exact Part 127 / Part 140 linkages remain subject to responsible Department Manager technical review.",
      "refreshPolicy": {
        "sourceCollectionId": "NCAA-NAMCATS-ALL-PAGES",
        "lastCheckedAt": "2026-07-28T20:46:37.914Z",
        "nextReconciliationDate": "2027-01-28",
        "nextExpertValidationDate": "2027-07-28",
        "eventDrivenReview": true,
        "reconciliationIntervalMonths": 6,
        "expertValidationIntervalMonths": 12,
        "sourceChangeState": "BASELINE_CAPTURED",
        "updateMode": "PROPOSE_DRAFT_ONLY",
        "documentCount": 58,
        "manifestPath": "docs/regulatory-sources/ncaa-namcats-manifest.json",
        "guardrails": [
          "A changed source creates a clause-impact review proposal; it never rewrites a published checklist.",
          "Source presence and extracted text do not establish applicability, legal interpretation, or expert validation.",
          "Existing audits remain pinned to their exact published checklist version."
        ]
      },
      "scopeRecommendation": {
        "id": "SCOPE-REC-AUD-2026-001-CABIN-001",
        "status": "ADVISORY_ONLY",
        "historyState": "INSUFFICIENT_FOR_DEFERRAL",
        "generatedAt": "2026-07-28T00:00:00Z",
        "signals": [
          "The current candidate screen scenario contains overdue Finding FND-CAB-2026-001 and pending Evidence review for PBE serviceability.",
          "No validated two-clean-audit history window is available for question-level deferral.",
          "The regulatory mapping and controlled NCAA ramp-inspection procedure remain under expert review."
        ],
        "guardrails": [
          "Mandatory, safety-critical, newly changed, overdue, open-Finding, and unknown-history controls cannot be automatically omitted.",
          "No recorded problem is not evidence of compliance.",
          "Any scope reduction requires Department Manager approval, a recorded rationale, and an audit trail.",
          "A full-scope inspection remains due at least annually even after a validated clean history is established."
        ],
        "questionRecommendations": [
          {
            "questionId": "CAB-GALLEY-001",
            "classification": "ROTATIONAL_SAMPLE",
            "rationale": "Keep the question in scope but use a representative sample unless aircraft condition, records, or prior findings indicate broader coverage.",
            "historyBasis": "No validated clean-history window supports deferral; sampling is the maximum current reduction.",
            "requiresManagerApproval": true
          },
          {
            "questionId": "CAB-LAV-001",
            "classification": "ROTATIONAL_SAMPLE",
            "rationale": "Sample lavatory equipment and placards, expanding to full coverage when fire-safety or maintenance signals are present.",
            "historyBasis": "No validated clean-history window supports deferral; safety escalation remains mandatory.",
            "requiresManagerApproval": true
          },
          {
            "questionId": "CAB-PAX-SEAT-001",
            "classification": "ROTATIONAL_SAMPLE",
            "rationale": "Use documented representative seat and restraint sampling, with immediate expansion when a defect is identified.",
            "historyBasis": "No validated clean-history window supports deferral; the sampling floor must be recorded.",
            "requiresManagerApproval": true
          },
          {
            "questionId": "CAB-EMEQ-PBE-001",
            "classification": "FOCUSED_FULL",
            "rationale": "Inspect the applicable PBE population and supporting records because the current scenario has an overdue related Finding and pending Evidence review.",
            "historyBasis": "FND-CAB-2026-001 is overdue and the latest PBE Evidence remains pending CAA review.",
            "requiresManagerApproval": true
          },
          {
            "questionId": "CAB-VID-CREW-SEAT-001",
            "classification": "MANDATORY_CORE",
            "rationale": "Crew restraint and exit-adjacent safety coverage stays in every inspection package.",
            "historyBasis": "Safety-critical status overrides clean-history or efficiency signals.",
            "requiresManagerApproval": true
          },
          {
            "questionId": "CAB-COCKPIT-GEN-001",
            "classification": "MANDATORY_CORE",
            "rationale": "Emergency-exit accessibility, markings, and visible cabin condition stay in every inspection package.",
            "historyBasis": "Safety-critical status overrides clean-history or efficiency signals.",
            "requiresManagerApproval": true
          }
        ]
      },
      "sources": [
        {
          "id": "ICAO-PQ-OPS-2024-R1.1",
          "title": "2024 USOAP CMA Protocol Questions - OPS",
          "sourceType": "ICAO_PQ",
          "version": "September 2024 Revision 1.1",
          "status": "SUPPLIED_WORKING_COPY",
          "locator": "04_OPS_2024 PQ Rev L FULL (en) - PQ 4.450",
          "url": null
        },
        {
          "id": "NCAA-CC-ANNEX6-PARTI-A610",
          "title": "Annex 6 Part I to NAMCAR compliance crosswalk",
          "sourceType": "ANNEX_CROSSWALK",
          "version": "Supplied working copy; edition not declared",
          "status": "SUPPLIED_WORKING_COPY",
          "locator": "CC.zip/CC/NAMB/Annex_NAMB_A610.docx - 4.2.2.2, 4.2.12.1, 6.2.2, 6.16, 8.1.1",
          "url": null
        },
        {
          "id": "AVIASURVEIL360-AUDIT-AREA-MAPPING-2026-07-27",
          "title": "AviaSurveil360 AI Regulatory Knowledge Mapping",
          "sourceType": "AUDIT_AREA_TAXONOMY",
          "version": "2026-07-27",
          "status": "SUPPLIED_WORKING_COPY",
          "locator": "AI Regulatory Mapping!A5:H5",
          "url": null
        },
        {
          "id": "NCAA-NAMCARS-LIBRARY",
          "title": "NCAA public NAMCARS download library",
          "sourceType": "NATIONAL_LIBRARY",
          "version": "Public library observed 2026-07-28",
          "status": "PUBLIC_REFERENCE",
          "locator": "NAMCARS public download index",
          "url": "https://www.ncaa.com.na/downloads.php?pagetitle=NAMCARS"
        },
        {
          "id": "NCAA-NAMCATS-LIBRARY",
          "title": "NCAA public NAMCATS download library",
          "sourceType": "NATIONAL_LIBRARY",
          "version": "All-pages baseline synchronized 2026-07-28 - 57 unique PDFs + linked index",
          "status": "PUBLIC_REFERENCE",
          "locator": "docs/regulatory-sources/ncaa-namcats-manifest.json",
          "url": "https://www.ncaa.com.na/downloads.php?pagetitle=NAMCATS"
        },
        {
          "id": "NCAA-OPS-SURVEILLANCE-PROCEDURE",
          "title": "NCAA Operations surveillance / ramp-inspection procedure",
          "sourceType": "CAA_PROCEDURE",
          "version": "Not supplied",
          "status": "SOURCE_GAP",
          "locator": "Controlled procedure/manual identity required",
          "url": null
        }
      ],
      "proposedQuestions": [
        {
          "id": "CAB-GALLEY-001",
          "prompt": "Are galley restraints and stowage areas serviceable and secure?",
          "verificationMethod": "Observe a representative sample and reconcile any defect with the approved MEL or defect-control record.",
          "evidenceExamples": ["Inspector observation", "Permitted photo", "MEL or defect-control record when applicable"],
          "whyIncluded": "PQ 4.450 calls for risk-based cabin/safety coverage; secure cabin stowage is a practical ramp-inspection verification point requiring expert confirmation against the operator and aircraft scope."
        },
        {
          "id": "CAB-LAV-001",
          "prompt": "Are lavatory safety equipment and placards present and serviceable?",
          "verificationMethod": "Observe the lavatory safety configuration and sample the applicable serviceability or inspection record.",
          "evidenceExamples": ["Inspector observation", "Placard and equipment condition", "Serviceability or maintenance record"],
          "whyIncluded": "Annex 6 Part I 6.2.2 and the mapped equipment clauses support cabin fire-safety and equipment checks within PQ 4.450 ramp coverage."
        },
        {
          "id": "CAB-PAX-SEAT-001",
          "prompt": "Are passenger seats, belts, and adjacent fittings serviceable?",
          "verificationMethod": "Sample passenger seats and restraints and reconcile exceptions with the approved defect-control process.",
          "evidenceExamples": ["Seat and restraint observation", "Permitted photo", "MEL or deferred-defect record"],
          "whyIncluded": "Annex 6 Part I 4.2.12.1 and 6.2.2 map passenger restraint and seat provisions to national clauses suitable for practical ramp verification."
        },
        {
          "id": "CAB-EMEQ-PBE-001",
          "prompt": "Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?",
          "verificationMethod": "Confirm installed position, accessibility, seal or expiry state, and the supporting serviceability record against the approved configuration.",
          "evidenceExamples": ["PBE position confirmation", "Expiry or seal check", "Serviceability or maintenance record"],
          "whyIncluded": "PQ 4.450 includes cabin/safety equipment within ramp inspection. The PBE-specific national clause and controlled inspection procedure still require expert validation."
        },
        {
          "id": "CAB-VID-CREW-SEAT-001",
          "prompt": "Are cabin information displays and crew seats serviceable?",
          "verificationMethod": "Observe the cabin information and crew-seat configuration, including harness condition and accessibility near required exits.",
          "evidenceExamples": ["Crew-seat and harness observation", "Cabin information display or placard check", "Defect record when applicable"],
          "whyIncluded": "Annex 6 Part I 6.16.1-6.16.3 maps cabin crew seat, harness, and exit-location requirements to NAMCAR 91/121 clauses."
        },
        {
          "id": "CAB-COCKPIT-GEN-001",
          "prompt": "Are cabin general condition and emergency exits satisfactory?",
          "verificationMethod": "Observe representative cabin condition, exit accessibility, markings, and visible safety-critical discrepancies.",
          "evidenceExamples": ["Inspector observation", "Exit accessibility and marking check", "Permitted photo or discrepancy record"],
          "whyIncluded": "PQ 4.450 explicitly includes cabin/safety in risk-based ramp inspections; Annex 6 passenger-information, equipment, and continuing-airworthiness provisions support the candidate decomposition."
        }
      ]
    }
  ]
}`
