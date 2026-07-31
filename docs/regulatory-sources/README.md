# Regulatory Source Context

This directory tracks source identity and refresh state for regulatory material
used by the `candidate-only` regulatory-knowledge workflow. Full source bytes
and extracted text are deliberately stored outside Git under:

```text
.local/aviasurveil360/regulatory-sources/
```

## NCAA NAMCATS complete public baseline

The bounded collection contains every public PDF listed across NCAA NAMCATS
pages 1-3 and the NAMCAR-NAMCATS index workbook linked from the library. One
PDF is listed on both pages 1 and 2; it is stored once with both listing pages
recorded.

- Tracked manifest:
  `ncaa-namcats-manifest.json`
- Public sources:
  `https://www.ncaa.com.na/downloads.php?pagetitle=NAMCATS&page=1`,
  `https://www.ncaa.com.na/downloads.php?pagetitle=NAMCATS&page=2`, and
  `https://www.ncaa.com.na/downloads.php?pagetitle=NAMCATS&page=3`
- Local files:
  `.local/aviasurveil360/regulatory-sources/ncaa/namcats/all-pages/files/`
- Local searchable text:
  `.local/aviasurveil360/regulatory-sources/ncaa/namcats/all-pages/text/`
- Resumable OCR checkpoints:
  `.local/aviasurveil360/regulatory-sources/ncaa/namcats/all-pages/ocr-checkpoints/`

Run or resume the bounded synchronization with:

```bash
node scripts/regulatory/sync-ncaa-namcats.mjs
node scripts/regulatory/sync-ncaa-namcats.mjs --verify-only
```

The synchronizer accepts only NCAA public `https` publication URLs, expects
the exact page-level listing counts, collapses the one repeated listing,
validates file signatures, records byte counts and SHA-256 digests, and reuses
unchanged valid local bytes.

Each PDF page with a searchable text layer uses Poppler extraction. Any
individual page without searchable text is rendered and processed locally with
Apple Vision OCR, including image-only pages inside otherwise searchable PDFs.
OCR checkpoints are stored per page so an interrupted run resumes instead of
starting the document again. The manifest distinguishes `EXTRACTED`,
`HYBRID_EXTRACTED`, `OCR_EXTRACTED`, and `OCR_NO_TEXT_DETECTED`.

## Service-provider catalog

The versioned [service-provider catalog v1](catalogs/service-provider-catalog.v1.json)
is the tracked 20-row source matrix for provider-scope identity, exact labels,
topics, raw responsible CAA units, aliases, typed targets, and normalization
status. Validate it with the canonical
[catalog schema](schemas/service-provider-catalog.schema.json). A raw owner
label containing `+` or `/` is `REVIEW_REQUIRED` and carries no inferred
semantic relationship or `approvalRequired` owner until NCAA confirms the
ownership relationship.

## Derived source context

Compact derived assessments under `derived/` preserve reviewed conclusions
without copying full regulatory text into Git. Each assessment must:

- bind every conclusion to an exact manifest source identity and SHA-256;
- retain the local full-text locator, PDF page, section, and a paraphrased
  evidence statement;
- separate direct, conditional, contextual, and missing-source dispositions;
- expose current-source, applicability, technical-validation, controlled-
  procedure, and publication gates; and
- remain invalidatable when a cited source hash changes.

The current OPS/AOC pilot assessment is:

- [Part 127 / Part 140 human-readable assessment](derived/ncaa-namcats-part-127-140-applicability.md)
- [Part 127 / Part 140 machine-readable record](derived/ncaa-namcats-part-127-140-applicability.json)

It classifies Part 127 as `OPERATION_TYPE_CONDITIONAL` and Part 140 as
`SYSTEM_LEVEL_APPLICABLE`. The existing derived records retain
`EXPERT_REVIEW_REQUIRED` as explicit readable legacy compatibility; new
workflow records use `TECHNICAL_REVIEW_REQUIRED` and require responsible
Department Manager review.
The 2025 Part 140 file is treated as the candidate current public reference and
the simultaneously listed 2021 file as a comparator until an NCAA source owner
confirms formal authority and supersession.

## Review cadence and state boundaries

- React to an authoritative publication or content-hash change as soon as it is
  observed.
- Reconcile the configured source collection at least every six months.
- Complete responsible Department Manager technical review at least annually.
- Treat `downloaded`, `text extracted`, `clause mapped`, `technically reviewed`,
  and `published in a checklist` as separate states.
- A source change proposes a clause-impact review and a new checklist Draft; it
  never changes a published checklist or an in-progress Audit.
- A page with no searchable text after OCR remains present and hash-verified
  but requires source-owner review before text-dependent mapping.

Self-assessment portals, authenticated forms, credentials, and restricted
sources are outside this source-intake path.
