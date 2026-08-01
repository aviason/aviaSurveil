# Form 048 Admin Review Request

**Package status:** `PENDING_ADMIN_REVIEW`

This is a source-backed, candidate-only review packet for the real Form 048
intake gate. It contains no AGA ZIP or source PDF bytes. It contains only the
28 question strings and provenance derived from the exact hash-bound Form 048
PDF, plus parser receipt/digest metadata; it contains no regulatory
citations, source-owner attestations, Department Manager decisions, or
publication approval.

The Turkish PDF `FORM_048_28_SORU_ADMIN_KARAR_FORMU_TR.pdf` now contains the
28 literal Form 048 protocol questions, their source PDF pages, AGA protocol
codes, and visible NAMCAR/NAMCATS references. Each card keeps the Admin
extraction decision blank (`NOT_SUPPLIED`) and leaves source mapping at
`SOURCE_MAPPING_REQUIRED`; the authenticated Admin must bind the decision to
the exact current intake packet proposal ID/digest. The derived question list
is also available as `FORM_048_28_SOURCE_QUESTIONS.json`. The Turkish handoff
draft is `FURKAN_MESSAGE_DRAFT.tr.md`.

## Verified inventory facts

- Archive SHA-256:
  `sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32`
- Archive size: `12227415` bytes
- Archive entries: `53` PDF entries (`1` register and `52` forms)
- Register SHA-256:
  `sha256:29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f`
- Form 048 file: `FSS-AGA-FORM-048.pdf`
- Form 048 SHA-256:
  `sha256:495aa7b0a1edca1ac5e874e6a63f50b47c6d207aa264cc390970a7db1acdc6e3`
- Form 048 facts: `9` pages and `28` visible protocol questions
- Visible/register title: `Checklist for the surveillance of an aerodrome`
- Embedded PDF metadata title: `Check list for inspection of the maintenance arrangements`
- Register inventory fact: `FSS-AGA-FORM-049` is absent and `035A` is present

The archive and file facts above are read-only receipt facts. They are not a
human identity decision and do not establish regulatory authority.

## What the current Admin must supply

1. An actor-bound identity decision selecting the human-readable Form 048
   identity while retaining the conflicting metadata title.
2. One actor-bound extraction decision for each of the 28 proposed question
   boundaries, including the exact packet proposal ID and digest, decision
   kind, reason, and any split/merge/transcription/exclusion details.
3. An explicit authorization to create one immutable
   `EXISTING_CHECKLIST_CANDIDATE` whose questions remain literal
   `SOURCE_MAPPING_REQUIRED`.
4. A named, bounded Phase 2 expansion authorization for additional candidate
   imports. This authorization must state the form/layout scope, batch limit,
   actor, decision ID, and that source mapping and publication are not
   authorized.

The product binds the supplied decisions to the exact immutable parser,
file, manifest, and packet digests. Blank, guessed, or copied IDs are
rejected fail-closed.

## Explicit non-authorizations

This packet does not request or grant regulatory source mapping, source-owner
attestation, Department Manager technical approval, publication, functional
assignment provisioning, deployment, release, or production readiness.
