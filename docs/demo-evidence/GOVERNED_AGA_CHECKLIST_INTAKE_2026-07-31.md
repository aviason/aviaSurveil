# Governed AGA Checklist Intake — Local Candidate Evidence

Date: 2026-07-31

Continuation preflight: 2026-08-01. The user explicitly authorized Task 9 /
Phase 2 continuation and selected the bounded controlled-intake option: the
exact external ZIP may be streamed by the Admin-only local intake service while
all `AGA_ZIP_PDF_V1`, authority, privacy, and fail-closed boundaries remain in
force. A current-worktree task-owned full profile reached API readiness with
migration `28` and healthy candidate dependencies. The archive receipt below
was created, but the real Form 048 Admin identity/28 boundary packet, immutable
candidate/source-gap Draft, and named expansion authorization were not
supplied, so no extraction decision, candidate, Draft, or expansion import ran.

This is metadata-only local evidence for the approved Governed AGA Checklist
Intake And Official-Source Authoring plan. It does not contain AGA ZIP/PDF
bytes, extracted text, screenshots, question content, or production evidence.
The product result is `candidate-only`; release is `release pending` and
`production-ready: not established`.

## External archive facts

The read-only verifier consumed the external path through
`AGA_CHECKLIST_ARCHIVE` and streamed the archive/hash/central-directory data.
It did not extract or copy anything into the repository.

- Archive filename: `AGA - Checklists and Form.zip`
- Archive bytes: `12,227,415`
- Archive SHA-256: `dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32`
- ZIP/PDF entries: `53` (one register and `52` forms)
- Total uncompressed bytes: `14,026,975`
- Register SHA-256: `29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f`
- Form 048 SHA-256: `495aa7b0a1edca1ac5e874e6a63f50b47c6d207aa264cc390970a7db1acdc6e3`
- Register facts: `FSS-AGA-FORM-001` through `FSS-AGA-FORM-053`, with `035A`
  present and `049` absent (`52` forms total)
- Form 048 planning facts: `9` pages and `28` visible protocol questions;
  visible/register identity conflicts with embedded PDF metadata and therefore
  remains `IDENTITY_REVIEW_REQUIRED`
- Read-only verifier receipt IDs: none; it remains metadata-only

These facts are inventory metadata only. They are not source authority,
applicability, mapping, technical approval, publication, or Audit eligibility.
The connected runtime checks below are disposable candidate evidence only and
do not establish production or external-owner authority.

The bounded Go intake slice performs raw ZIP preflight before opening entries:
ZIP64 and multi-disk metadata, data descriptors, local/central filename and
flag/method/CRC/size mismatches, overlapping local ranges, unaccounted gaps,
and trailing bytes are rejected. An accepted local receipt remains
`PROCESSING` with an `ARCHIVE_VALIDATE` phase receipt; scan/parser/finalizer
receipts are not fabricated when their runtime dependencies are unavailable.

## Controlled intake receipt — 2026-08-01

The latest successful exact-archive run streamed the external archive through
the local Admin-only HTTP intake service using the server-owned canonical test
`USR-ADMIN-ADA`. The request never wrote ZIP/PDF bytes or extracted text to the
repository or to this evidence file.
The receipt was the first multipart part; an independent mismatched
header/receipt request returned `400` before any archive-content read.

- HTTP result: `201 Created`
- Import batch: `import-f0b57aeb10f8383c`
- Archive SHA-256: `sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32`
- Manifest digest: `sha256:0e40137deddf57b01254a164b4529400eff3f434c07050c478be628e69264fec`
- Batch status: `PROCESSING`
- Archive validation receipt: `import-f0b57aeb10f8383c-archive-validate`
- Receipt outcome/policy: `SUCCEEDED` / `AGA_ZIP_PDF_V1`
- Receipt payload: `12,227,415` archive bytes, `53` entries, `53` PDFs, `0`
  directories
- Controlled runtime command: `./scripts/test-http-profile.sh` focused
  governed-checklist-intake profile; exit `0`; task-owned cleanup completed

Post-review hardening verification is also `verified locally`: Gate-0 Node
tests `5/5`, focused Go boundary tests `7/7`, local profile contract `14/14`,
inventory `gate0` with `11` artifacts, harness smoke, syntax, diff, and
untracked-document whitespace all passed. The phased `task9` and `final`
inventory commands intentionally exited `2` for the unchanged external
Form 048/Admin decision, candidate/source-gap Draft, and expansion
authorization blockers.

This receipt proves only bounded archive receipt/inventory. It does not prove
ClamAV/PDF parser completion, register parse, identity resolution, extraction
decisions, candidate import, source authority, source mapping, Draft creation,
technical approval, publication, or Audit eligibility. The batch therefore
remains `PROCESSING` with `SECURITY_PHASES_PENDING`.

## Continuation verification record — 2026-08-01

The following fresh continuation checks were run locally after explicit
authorization to resume Task 9. `verified locally` means the named local
contract/mechanism check passed; it does not establish production or external
decision authority.

| Command | Result |
| --- | --- |
| `node scripts/verify-governed-checklist-test-inventory.mjs --phase gate0` | exit `0`; `11` required artifacts |
| `node --test tests/governed-checklist-intake-plan-contract.test.mjs tests/governed-checklist-intake-security.test.mjs tests/governed-checklist-discriminator-contract.test.mjs` | exit `0`; `5/5` passed |
| `AGA_CHECKLIST_ARCHIVE=… node --test tests/aga-checklist-archive-inventory.test.mjs` | exit `0`; `1/1` passed; metadata-only read |
| connected task-owned integration binary against PostgreSQL | exit `0`; `4/4` passed (synthetic Form 048 mechanism, migration 28, migration-path, and Task-9 authority/privacy boundary) |
| `GOCACHE=… go -C apps/api test ./tests/integration -tags canonicaltest -run '^TestAGACandidateExpansion$' -count=1 -v` | exit `0`; one explicit `blocked` skip for the absent real Admin packet/candidate/source-gap Draft/named expansion authorization |
| task-owned full-profile readiness | `verified locally`; `/health/ready` ready, schema migration `28`, API/worker/scheduler/PostgreSQL/MinIO/ClamAV/Gotenberg healthy; lifecycle-created local Admin harness identity was configured with `emailVerified=true` and `CONFIGURE_TOTP` required action |
| `node scripts/verify-governed-checklist-test-inventory.mjs --phase task9` | exit `2`; real Form 048 Admin identity/28 boundary packet, immutable candidate/source-gap Draft, and named expansion authorization remain blocked |
| `node scripts/verify-governed-checklist-test-inventory.mjs --phase final` | exit `2`; all inventory and runner discovery passed, then the same explicit blocker was reported |
| focused Go governed packages | exit `0`; checklistintake, checklistgovernance, HTTP API, regulatory, and five identity authority/privacy tests passed; unrelated httptest listener tests remain sandbox-blocked |
| governed React tests | exit `0`; `9` files, `10/10` passed |
| HTTP parity test | exit `0`; `2/2` passed |
| `npm --prefix apps/web run typecheck && npm --prefix apps/web run build:demo && npm --prefix apps/web run build:http` | exit `0`; typecheck and both builds passed |
| `node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http` | exit `0`; `148` files / `179` inputs |
| `./scripts/test-governed-checklist-intake-profile.sh --security-only` | exit `0`; `23/23` security/policy tests and cleanup passed |
| `node tests/harness-docs-smoke.test.js`, `git diff --check`, explicit plan/evidence whitespace check | exit `0`; clean |
| task-owned runtime cleanup assertion | exit `0`; no task-owned residue |
| independent read-only continuation review | `APPROVED`; no Critical, Important, or Minor finding |

The source-gap transport digest received an additional fresh RED/GREEN repair
after the continuation record above. A Go invariant first failed because the
literal `NOT_AVAILABLE` review projection changed the candidate digest from
`sha256:1c1cee0ca367092d54c80d1fce792fe3613c52bfec1aa64e3e708e390c1a0ba6`
to `sha256:81bb76432c4f06c33085d4f73738f761626d26817e9d9e647674b6ae46c8a595`.
The smallest repair removes that server-derived source-gap projection from
Go canonicalization and the persisted output artifact. The focused Go
invariant and React parity command then passed (`24/24`), followed by fresh
connected governed-checklist HTTP (`2/2`) and regulatory-source-refresh HTTP
(`2/2`) profiles with task-owned cleanup. No source mapping, publication, or
additional AGA form was imported.

The first metadata-only Admin review packet at
`deliverables/FURKAN_FORM_048_ADMIN_REVIEW_REQUEST_2026-08-01.zip` was
superseded in place by the source-backed handoff recorded below. It has not
been sent externally; the archive still contains no raw AGA ZIP/PDF bytes,
source authority, approval, or production evidence.

An independent final read-only review of this current worktree returned
`APPROVED` with no Critical, Important, or Minor findings. It confirmed the
narrow source-gap digest projection, strict validation, exact Task-9 blockers,
and preserved role, authority, privacy, publication, and evidence-label
boundaries.

## Source-backed Form 048 question handoff — 2026-08-01

Under the user's explicit continuation authorization, the bounded local
source-read path streamed the exact archive, selected only
`FSS-AGA-FORM-048.pdf` into mode-0600 task-owned temporary storage, ran the
pinned Poppler `pdftotext -layout` extractor, and removed the temporary source
bytes after generation. No raw ZIP/PDF bytes were copied into Git.

- Archive: `12,227,415` bytes,
  `sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32`
- Form 048: `9` pages,
  `sha256:495aa7b0a1edca1ac5e874e6a63f50b47c6d207aa264cc390970a7db1acdc6e3`
- Extracted questions: `28/28`, each with protocol code, source page/line,
  visible NAMCAR/NAMCATS references, and literal source wording
- Parser receipt: `sha256:adc3a19a56109a44a6e48e9effe4d285a803e63feba836298cf313cdd6039505`
- Derived proposal packet digest:
  `sha256:4b3a07ea20af11e762a93dc1ae0223400c307f6a0af83d9471769f11082d2a4b`
- PDF artifact: `11` A4 pages; source-packet test `2/2`; package manifest
  `8/8`; ZIP integrity `unzip -t` passed; ZIP SHA-256:
  `e9ac47ecfd3393e81a43f028343b30cdc7142791050977ca8957f3cdde3dd998`
- Decision state: all 28 are `NOT_SUPPLIED`; source mapping remains
  `SOURCE_MAPPING_REQUIRED`; proposal IDs are
  `PENDING_CURRENT_SERVER_PACKET_BINDING`
- No identity resolution, candidate, Draft, source-authority decision,
  publication, functional assignment, or Phase 2 form import was created.

The current Admin must review the literal questions and submit actor-bound
decisions through the exact current intake packet. The derived handoff is
candidate-only review input, not regulatory authority or production evidence.

The repaired manifest and embedded ZIP manifest were independently rechecked
(`8/8` and `unzip -t`), and the independent read-only Gate 0 review returned
`APPROVED` with no findings. The final phased inventory remains intentionally
`BLOCKED` (`exit 2`) until the external Admin identity/28 decisions, immutable
candidate/source-gap Draft, and named expansion authorization are supplied.

## Initial verification record — 2026-07-31

These historical checks remain part of the accepted local evidence. `verified
locally` means the named local contract/mechanism check passed; it does not
establish production or external decision authority.

| Command | Result |
| --- | --- |
| `node scripts/verify-governed-checklist-test-inventory.mjs --phase gate0` | exit `0`; `11` required artifacts |
| `AGA_CHECKLIST_ARCHIVE=… node --test tests/governed-checklist-intake-plan-contract.test.mjs tests/governed-checklist-intake-security.test.mjs tests/aga-checklist-archive-inventory.test.mjs tests/governed-checklist-discriminator-contract.test.mjs` | exit `0`; `6/6` passed |
| `node --test` governed OpenAPI/transport suite | exit `0`; `9/9` passed |
| `./scripts/check-contracts.sh` | exit `0`; `16/16` contract tests and generated drift clean |
| `node scripts/lint-openapi.mjs` | exit `0`; lint clean |
| Focused Go candidate/intake/regulatory/HTTP packages | exit `0`; all listed packages passed |
| Focused React/mock parity command | exit `0`; `11` files, `24/24` passed |
| Task 5 semantic parity | exit `0`; `3/3` passed |
| Task 6 manager/artifact parity | exit `0`; `16/16` passed |
| HTTP contract parity | exit `0`; `2` files, `5/5` passed |
| `npm --prefix apps/web run typecheck` | exit `0` |
| `npm --prefix apps/web run build:demo` and `build:http` | exit `0` for both |
| `node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http` | exit `0`; `148` files / `179` inputs |
| Root legacy checklist/boundary smoke | exit `0`; `5/5` passed |
| `./scripts/test-governed-checklist-intake-profile.sh --security-only` | exit `0`; inventory `18` artifacts, Node security/policy `23/23` passed, cleanup clean |
| `node tests/harness-docs-smoke.test.js` | exit `0`; `harness-docs-smoke: ok` |
| `node scripts/verify-governed-checklist-test-inventory.mjs --phase task8` | exit `0`; `42` required artifacts |
| `node scripts/verify-governed-checklist-test-inventory.mjs --phase task9` | exit `2`; real Form 048 Admin identity/28 boundary packet, immutable candidate/source-gap Draft, and named expansion authorization blocker |
| task-owned connected integration binary, four focused tests | exit `0`; `4/4` passed against migration-28 PostgreSQL (synthetic Form 048 mechanism, migration, migration-path, and Task-9 authority/privacy boundary) |
| `GOCACHE=… go -C apps/api test ./tests/integration -tags canonicaltest -run '^TestAGACandidateExpansion$' -count=1 -v` | exit `0`; test explicitly skipped as `blocked` because the real Form 048 Admin identity/28 boundary packet, immutable candidate/source-gap Draft, and named expansion authorization are absent |
| `node scripts/verify-governed-checklist-test-inventory.mjs --phase final` | exit `2`; all required inventory and runner discovery passed, then the same explicit blocker was reported |
| Focused browser runner discovery from `apps/web` | exit `0`; Vitest `2` cases, Playwright mock `1`, HTTP `1` |

The canonical projection TDD follow-up first ran the focused Vitest selector
with exit `1` (`3` failed, `6` skipped) against the intentionally missing
projection scrub, then the full parity file ran with exit `0` (`9/9` passed).
It covers candidate, import, and edit digest invariance for mapping-review
projections and the literal `SOURCE_MAPPING_REQUIRED`/`NOT_AVAILABLE`
technical projection.

The migration/integration selector exited `0` with `6` mechanism tests passed
and `1` Task-9 expansion test skipped with its explicit blocker. The broader Go
application command exited `1` only because unrelated identity tests attempt to
bind `httptest` listeners (`operation not permitted` in this sandbox); the
focused governed packages passed. The parser/scanner aggregate likewise has
the existing fake-clamd listener bind restriction; live scanner/parser
qualification is therefore `blocked`, not a product-pass claim.

The initial Gate-0 RED was exit `1` with `1` pass and `4` expected contract /
inventory failures. The repaired Gate-0 GREEN was exit `0` with `5/5` passed.
The initial independent Gate-0 review and inventory-tightening review, followed
by the post-repair digest/UI review, returned `APPROVED` with no Critical,
Important, or Minor finding. The final review confirmed candidate/import/edit
and source-gap digest invariance, the authority-neutral queue/projection
decision, and no regression to the strict archive/process/receipt boundary.

## Blocked external dependencies

The following remain intentionally `blocked` and must not be inferred from
local fixtures or skipped runtime tests:

- real Form 048 authenticated Admin identity and all boundary decisions;
- real source-owner/source-set authority, currentness, applicability, mapping,
  and controlled-procedure decisions;
- real functional-assignment grant/revoke actor, anti-self-authorization rule,
  and evidence contract;
- responsible Department Manager technical-review and separate publication
  decisions;
- real Form 048 current authenticated Admin identity and all 28 boundary
  decisions, immutable candidate, visible source-gap Draft, and named
  expansion authorization for Phase 2 expansion;
- disposable PostgreSQL/object-store crash-recovery profile;
- real-archive MinIO/ClamAV/isolated Poppler parser qualification,
  browser/live HTTP scenarios, and process/container qualification evidence.

No external system was contacted, and no archive source bytes were added to
Git.
