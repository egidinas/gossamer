# Gossamer Backlog

Items in priority order. Each item has a status, description, and acceptance criteria.

---

## Shared Loom / Gossamer backlog
**Status:** active
Cross-repo contract, fixture, evidence, and agent-tooling work is synchronized in
[`shared_loom_gossamer_backlog.md`](shared_loom_gossamer_backlog.md). Gossamer
may adopt sibling-system patterns only at the public-safe contract/workflow
level. Do not import live adapters, private identifiers, host details,
credentials, protocol databases, captures, or hardware-specific procedures.

---

## GOSS-00 · Finish public-safe shared-backlog slice
**Status:** implementation done; commit/review pending
The current working tree contains source-ownership vocabulary, backend-authored
source tree fixtures, graph-wall manifest plumbing, and UI/API contract work
inspired by the shared backlog. Integrate that slice before starting new visual
polish so the public-safe semantic model is coherent.
**Fix:** review, verify, and commit `owner_mode`, `use`, `format_preference`,
`discovery_path`, source tree config, graph-wall manifest API, and source
catalogue rendering as one focused public-safe slice.
**AC:** `go test ./internal/contracts ./internal/synthetic` passes;
`cd web && npm run test:contracts` passes; `cd web && npm run test:clean-room`
passes (public-safe clean-room checklist); public fixtures/source/docs pass a clean-room
scan; generated binaries and build artifacts are not tracked.

---

## GOSS-01 · Marker label overlap
**Status:** implemented; commit/review pending
Dense test phases on the primary FAT/TVac card render diagonal phase labels that collide at normal zoom levels. Labels become unreadable at 4-cycle or 8-cycle density.  
**Fix:** render short labels (≤8 chars) by default; suppress overlapping neighbours; show full label on hover via tooltip.  
**Current implementation:** event rail markers keep the marker dot and hover
title visible, render labels through `shortGateLabel`, and suppress only
neighbouring labels that would crowd the rail. `npm run test:contracts` guards
the truncation, label suppression, and hover-title behavior.
**AC:** no two visible labels overlap at 1440p full-zoom on thermal_acceptance_fat and tvac_qualification.

---

## GOSS-02 · 4K card height under-utilisation
**Status:** implemented; commit/review pending
At 3840px the operator center lanes are ~352px each, leaving large dead space below the fourth lane. The `clamp` ceiling is capped at 320px which was designed for 1080p headroom.  
**Fix:** raise the upper bound of the `clamp` for `command_center_fat` lanes at wide viewports so all four lanes together fill ~85% of the viewport height.  
**Current implementation:** command-center lane height caps are raised for wide
viewports, and `npm run test:browser` includes a 3840x2160 command-center smoke
check that requires the first four visible lanes to occupy at least 80% of the
viewport height.
**AC:** at 3840×2160 the four command-center lanes collectively occupy ≥80% of viewport height.

---

## GOSS-03 · functional_events card unbounded height
**Status:** implemented; commit/review pending
The `functional_events` event-rail card grows to 1271–1313px on desktop because the event rail has no height cap. It dwarfs every other card and breaks section rhythm.  
**Fix:** cap event-rail cards at `max-height: 480px` with `overflow-y: auto` inside the plot shell; or cap the swimlane row count at render time.  
**Current implementation:** event-rail cards have a bounded resize height and a
plot-shell max height with vertical scrolling. `npm run test:contracts` guards
the cap.
**AC:** functional_events card ≤ 500px at 1440p; content scrollable if truncated.

---

## GOSS-04 · Mobile graph horizontal scroll
**Status:** implemented; commit/review pending
Acceptance FAT and Qualification TVac overflow by ~243px on mobile (390px viewport). The time axis and right-side y-labels are clipped. Currently noted as acceptable but a scroll container would make the graph pannable.  
**Fix:** wrap `.operator-wall-scrollframe` content in an `overflow-x: auto` scroll container at narrow viewports; or constrain the shared time axis and label-rail to the visible width.  
**Current implementation:** mobile FAT/TVac graph walls get an explicit
horizontal scrollframe, stable scroll gutter, and minimum graph width.
`npm run test:browser` now visits the acceptance and qualification routes on
mobile and verifies the scrollframe behavior using fresh local ports.
**AC:** on 390px viewport, acceptance and tvac graphs are either fully contained or explicitly horizontally scrollable with no clipping.

---

## GOSS-05 · Remove one-off scripts from repo root
**Status:** implemented; commit/review pending
`do_gofmt.sh`, `do_refactor.py`, `refactor_*.py`, `fix_recover.py`, `safego_refactor.py`, and similar one-shot files are sitting untracked in the repo root. They pollute `git status` and are confusing to anyone cloning the repo.  
**Fix:** delete all of them (they were single-use refactor aids, not ongoing tools).  
**Current implementation:** the repo root no longer contains untracked `*.py`
or ad-hoc `*.sh` files.
**AC:** `git status` shows no untracked `*.py` or ad-hoc `*.sh` files in the repo root.

---

## GOSS-06 · Commit outstanding fixture changes
**Status:** implemented; commit/review pending
`thermal_acceptance_fat` and `tvac_qualification` tiles, manifests, and telemetry archives are modified but not committed. The working tree is dirty against the deployed state.  
**Fix:** verify that fixtures were regenerated from current generators/contracts and that existing deltas are intended before staging for commit.
**Current verification:** `go run ./cmd/gossamer-fixtures` currently reports no generated-file changes; `go test ./internal/contracts ./internal/synthetic ./internal/api` and `cd web && npm run test:contracts` pass.
**AC:** the current fixture/contract set is coherent with generator output and can be marked implemented pending commit/review.

---

## GOSS-07 · Shared backend semantics checklist
**Status:** implemented; commit/review pending
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
**Status:** implemented; commit/review pending
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
**Status:** Gossamer benchmark implemented; Loom counterpart pending
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
**Status:** implementation done; review pending
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
**Status:** implemented; commit/review pending
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
