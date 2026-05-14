# Gossamer Backlog

Items in priority order. Each item has a status, description, and acceptance criteria.

---

## SignalForge / Gossamer backlog boundary
**Status:** active
Cross-repo neutral primitives live in the public SignalForge module and are
consumed by Gossamer through `github.com/egidinas/signalforge/*`. Gossamer may
adopt sibling-system patterns only at the public-safe contract/workflow level.
Do not import live adapters, private identifiers, host details, credentials,
protocol databases, captures, or hardware-specific procedures.
Public-variant readiness gates are tracked in
`docs/public_variant_readiness.md`. Cross-project ideas that should not be lost
are tracked in `docs/cross_project_transfer_register.md` and must be promoted
through public-safe SignalForge contracts before becoming reusable code here.

---

## GOSS-00 · Finish public-safe shared-backlog slice
**Status:** done — verified 2026-05-14 (`go test ./...` and `npm run test:contracts` pass; `owner_mode`, `use`, `format_preference`, `discovery_path`, graph-wall manifest API, and source catalogue rendering confirmed in committed code)
The current working tree contains source-ownership vocabulary, backend-authored
source tree fixtures, graph-wall manifest plumbing, and UI/API contract work
inspired by the shared backlog. Integrate that slice before starting new visual
polish so the public-safe semantic model is coherent.
**Fix:** review, verify, and commit `owner_mode`, `use`, `format_preference`,
`discovery_path`, source tree config, graph-wall manifest API, and source
catalogue rendering as one focused public-safe slice.
**AC:** `go test ./...` passes; `cd web && npm run test:contracts` passes;
public fixtures/source/docs pass the clean-room scan; generated binaries and
build artifacts are not tracked.

---

## GOSS-01 · Marker label overlap
**Status:** done — verified 2026-05-14 (`shortGateLabel`, label suppression, and hover-title present in markers.ts and uPlotAdapter.ts; `npm run test:contracts` passes)
Dense test phases on the primary FAT/TVac card render diagonal phase labels that collide at normal zoom levels. Labels become unreadable at 4-cycle or 8-cycle density.  
**Fix:** render short labels (≤8 chars) by default; suppress overlapping neighbours; show full label on hover via tooltip.  
**Current implementation:** event rail markers keep the marker dot and hover
title visible, render labels through `shortGateLabel`, and suppress only
neighbouring labels that would crowd the rail. `npm run test:contracts` guards
the truncation, label suppression, and hover-title behavior.
**AC:** no two visible labels overlap at 1440p full-zoom on thermal_acceptance_fat and tvac_qualification.

---

## GOSS-02 · 4K card height under-utilisation
**Status:** done — verified 2026-05-14 (wide-viewport lane height caps present in committed CSS; `npm run test:contracts` passes)
At 3840px the operator center lanes are ~352px each, leaving large dead space below the fourth lane. The `clamp` ceiling is capped at 320px which was designed for 1080p headroom.  
**Fix:** raise the upper bound of the `clamp` for `command_center_fat` lanes at wide viewports so all four lanes together fill ~85% of the viewport height.  
**Current implementation:** command-center lane height caps are raised for wide
viewports, and `npm run test:browser` includes a 3840x2160 command-center smoke
check that requires the first four visible lanes to occupy at least 80% of the
viewport height.
**AC:** at 3840×2160 the four command-center lanes collectively occupy ≥80% of viewport height.

---

## GOSS-03 · functional_events card unbounded height
**Status:** done — verified 2026-05-14 (event-rail bounded max-height with overflow-y scroll present in OperatorGraphWall.tsx and views.css; `npm run test:contracts` passes)
The `functional_events` event-rail card grows to 1271–1313px on desktop because the event rail has no height cap. It dwarfs every other card and breaks section rhythm.  
**Fix:** cap event-rail cards at `max-height: 480px` with `overflow-y: auto` inside the plot shell; or cap the swimlane row count at render time.  
**Current implementation:** event-rail cards have a bounded resize height and a
plot-shell max height with vertical scrolling. `npm run test:contracts` guards
the cap.
**AC:** functional_events card ≤ 500px at 1440p; content scrollable if truncated.

---

## GOSS-04 · Mobile graph horizontal scroll
**Status:** done — verified 2026-05-14 (scrollframe, overflow-x auto, and minimum graph width present in OperatorGraphWall.tsx and responsive.css; `npm run test:contracts` passes)
Acceptance FAT and Qualification TVac overflow by ~243px on mobile (390px viewport). The time axis and right-side y-labels are clipped. Currently noted as acceptable but a scroll container would make the graph pannable.  
**Fix:** wrap `.operator-wall-scrollframe` content in an `overflow-x: auto` scroll container at narrow viewports; or constrain the shared time axis and label-rail to the visible width.  
**Current implementation:** mobile FAT/TVac graph walls get an explicit
horizontal scrollframe, stable scroll gutter, and minimum graph width.
`npm run test:browser` now visits the acceptance and qualification routes on
mobile and verifies the scrollframe behavior using fresh local ports.
**AC:** on 390px viewport, acceptance and tvac graphs are either fully contained or explicitly horizontally scrollable with no clipping.

---

## GOSS-05 · Remove one-off scripts from repo root
**Status:** done — verified 2026-05-14 (no untracked *.py or ad-hoc *.sh files in repo root; `git status` clean)
`do_gofmt.sh`, `do_refactor.py`, `refactor_*.py`, `fix_recover.py`, `safego_refactor.py`, and similar one-shot files are sitting untracked in the repo root. They pollute `git status` and are confusing to anyone cloning the repo.  
**Fix:** delete all of them (they were single-use refactor aids, not ongoing tools).  
**Current implementation:** the repo root no longer contains untracked `*.py`
or ad-hoc `*.sh` files.
**AC:** `git status` shows no untracked `*.py` or ad-hoc `*.sh` files in the repo root.

---

## GOSS-06 · Commit outstanding fixture changes
**Status:** done — verified 2026-05-14 (`go test ./...` and `npm run test:contracts` pass; working tree clean; fixtures coherent with generator output)
`thermal_acceptance_fat` and `tvac_qualification` tiles, manifests, and telemetry archives are modified but not committed. The working tree is dirty against the deployed state.  
**Fix:** verify that fixtures were regenerated from current generators/contracts and that existing deltas are intended before staging for commit.
**Current verification:** `go run ./cmd/gossamer-fixtures`, `go test ./...`,
and `cd web && npm run test:contracts` pass against the shared contract and
fixture boundary.
**AC:** the current fixture/contract set is coherent with generator output and can be marked implemented pending commit/review.

---

## GOSS-07 · Shared backend semantics checklist
**Status:** done — verified 2026-05-14 (`docs/backend_semantics_checklist.md` present; `npm run test:contracts` guards required terms)
Gossamer and its sibling system now share a discipline: the backend owns semantic
meaning, while the browser renders already-classified contracts. This needs a
small checklist so UI work does not drift back into local inference.
**Fix:** add or reference a checklist covering backend-owned role, authority,
freshness, provenance, source/target IDs, fixture/live distinction, and
clean-room language.
**Current implementation:** `docs/backend_semantics_checklist.md` defines the
public-safe review checklist, and `npm run test:contracts` fails if the required
backend-semantics terms are removed.
**AC:** source catalogue, graph wall, bus tap, command authority, and evidence
report changes cite the checklist during review.

---

## GOSS-08 · Evidence report cross-check with sibling contract
**Status:** done — verified 2026-05-14 (report fixtures include fixture status, requirement references, source provenance, anomaly summaries, command authority, and review notes; `npm run test:contracts` passes)
Gossamer's reports are strong as deterministic campaign evidence, while the
sibling system needs sharper language around fixture proof versus live proof.
Gossamer should keep its public-safe report vocabulary but align the field
concepts where useful.
**Fix:** compare report fields for requirement reference, source provenance,
anomaly summary, command authority, generated-at time, fixture/mock status, and
review notes. Add only generic public-safe wording.
**Current implementation:** report fixtures now include fixture status,
requirement references, source provenance, anomaly summaries, command
authority, and review notes. `npm run test:contracts` cross-checks these fields
against campaign requirements, report sources, and anomaly records.
**AC:** at least one report fixture demonstrates traceability without implying
hardware validation.

---

## GOSS-09 · Agent context codec fixture pack
**Status:** Gossamer side done — verified 2026-05-14 (`fixtures/public/agent_context_codec_benchmark.json` present; `npm run test:contracts` validates compact encoding, round-trip, and required fields). Loom counterpart (S-LG-06) still pending.
Large source catalogues, graph manifests, report JSON, and backlog slices are
expensive to pass through coding agents repeatedly. Gossamer should provide a
representative public-safe fixture pack for measuring compact, schema-aware
agent encodings.
**Fix:** choose representative Gossamer payloads and record exact round-trip and
token-count requirements for any compact format. Keep canonical JSON as the
source of truth.
**Current implementation:** `fixtures/public/agent_context_codec_benchmark.json`
contains source-catalogue and command-event-ring slices. `npm run
test:contracts` now validates the compact table encoding, measured byte/token
estimates, required identity/provenance/authority/fixture fields, and canonical
round-trip behavior.
**AC:** compact encoding preserves provenance, authority, fixture/mock status,
source IDs, and report references after round-trip through canonical JSON.
Shared `S-LG-06` can close only after Loom records a matching measured payload.

---

## GOSS-10 · Discovery tree and graph assignment UX comparison
**Status:** done — verified 2026-05-14 (shared/demo-only/live-only classification present in source_catalogue.json, source_tree_config.json, and graph_wall_manifest.json)
The source catalogue is converging on a collapsible, backend-authored tree model,
with source grouping from fixture semantics and static discovery-path provenance.
**Current implementation:** the shared backlog now records the classification for
discovery-tree and graph-assignment behavior so each Gossamer item can state:

- shared: adopted patterns with deterministic fixtures and backend-owned semantics
- demo-only: synthetic-only features that are intentionally not live-capable
- live-only: intentionally absent in Gossamer because they require live command,
  transport, or hardware orchestration

The shared matrix now references both catalogue and graph assignment details from
`fixtures/public/source_catalogue.json`, `fixtures/public/source_tree_config.json`,
`fixtures/public/graph_wall_manifest.json`, and related graph-wall contracts.
**AC:** any backlog close-out or review note for source catalogue and graph-wall
changes includes an explicit shared/demo-only/live-only classification.

---

## GOSS-11 · Remove live-system bus vocabulary from public UI copy
**Status:** done — verified 2026-05-14 (App.tsx boot copy uses generic terms; `npm run test:contracts` guards public-facing path; documented legacy identifier exceptions retained per cleanup rule)
The clean-room scan currently flags `web/src/App.tsx` strings that describe a
public demo bus as `CAN/TMTC`. That wording is too close to live-system
vocabulary for a public-safe demonstrator.
**Fix:** replace those UI strings with generic public-safe terms such as
`command/telemetry bus`, `supervisor bus`, or another fixture-owned bus name.
Keep actual data contracts deterministic and generic.
Cleanup rule: this cleanup targets user-facing labels/copy/comments. Deterministic
identifiers that must stay stable (for example `tvac_tmtc_primary`,
`tvac_tmtc_backup`, `archive_bus`, and transport role IDs) can retain legacy
fragments only when changing them would break contract consumers.
**Current implementation:** `App.tsx` boot-copy no longer uses `CAN/TMTC`, and
`npm run test:contracts` now guards the broader public-facing path. The wider
fixture/internal/docs vocabulary scan pass is complete for labels, names, and
documentation copy; only the documented legacy identifier cases remain.
**AC:** a clean-room vocabulary scan over `fixtures/public`, `internal`,
`web/src`, and public docs no longer finds private/live-system bus terms except
inside backlog items that explicitly document the cleanup rule.

---

## GOSS-12 · [MT-GOSS-01] Fictional scanner/setup confirmation demo
**Status:** planned 2026-05-15 — harvest candidate from mynaric_telemetry before archival.
Scanner identity/setup flow and uPlot/graph surface lessons from mynaric_telemetry are
candidates for a fictional public demo in Gossamer.
**Fix:** recreate scanner confirmation and setup step patterns as deterministic fictional
fixtures with neutral placeholder identities; do not copy real testbed labels, captures,
hostnames, or payloads. Relevant patterns: scan equipment identity, confirm setup steps,
feed requirement progress, create auditable event trail.
**AC:** Gossamer has a fictional scanner confirmation demo using placeholder identities;
`npm run test:contracts` and clean-room scan pass; no real testbed labels or mynaric
identifiers appear in fixtures, source, or docs.

---

## GOSS-13 · [KV-SF-01 / MT-SF-01] Watch for SignalForge neutral CAN and archive harvest candidates
**Status:** planned 2026-05-15 — future SignalForge extractions may produce Gossamer-consumable primitives.
When SignalForge receives neutral CAN adapter interface, capture/reduction vocabulary,
artifact bundle metadata, Arrow/LTTB tile contract, or archive/replay primitives from
the kvaser-dual-bridge or mynaric_telemetry harvest, Gossamer should evaluate consuming
them through the public SignalForge module.
**Fix:** review each SignalForge audit note for KV-SF-01 / MT-SF-01 rows; adopt any
primitives that improve the Gossamer demo without adding private dependencies.
**AC:** Gossamer consumes only public SignalForge versions; no local `replace` directives;
clean-room scan passes after any adoption.
