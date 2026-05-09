# Shared Loom / Gossamer Backlog

Date: 2026-05-09

This backlog synchronizes cross-pollination work between Loom and Gossamer. It
tracks shared contracts, vocabulary, fixture discipline, review checklists, and
agent tooling. It does not move live adapters, private node details, credentials,
or hardware-specific behavior between repositories.

## Ownership Boundary

- Loom is the real local/distributed testbed system. It owns live node
  discovery, controller sessions, command authority, rich decoded subscription
  targets, graph-wall assignment, provenance, and operational proof.
- Gossamer is the public-safe deterministic demonstrator. It owns synthetic
  fixtures, generic topology, backend-owned semantics, mocked authority, and
  offline evidence/report workflows.
- Shared work must stay at the contract and workflow level unless a private fork
  explicitly opts into live integrations.

## Current State Snapshot

- At the time of this review, Loom's active queue is
  `docs/backlog/goals.jsonl`; pairwise review lives in
  `docs/backlog/loom_gossamer_pairwise_review.md`.
- Gossamer `master` has implemented but still dirty/unintegrated changes for
  source ownership vocabulary, a backend-authored discovery tree, graph-wall
  assignment fixtures, and UI/API contract plumbing. Preserve that dirty work
  and integrate it as a focused public-safe backlog slice.
- The same shared backlog file is mirrored in both repos so agents can start
  from either checkout without guessing which side owns the next step.

## Rules

- Backend owns semantics. Browser code renders authority, source role,
  provenance, freshness, and target format; it must not infer those from names.
- Canonical JSON, JSONL, Arrow, HDF5, or existing repo-native fixtures remain
  authoritative. Compact agent encodings are derived convenience forms only.
- Fixture evidence is not live proof. Loom live packets still need observed
  node status, command results, and transport provenance.
- Gossamer imports patterns only after clean-room review. No real adapter names,
  node names, private networks, credentials, protocol databases, captures, or
  private procedures enter the public-safe repo.
- Shared UI patterns must keep their different operating modes visible:
  live/authoritative in Loom, deterministic/mock in Gossamer.

## Shared Items

### S-LG-01 - Mirror shared backlog and backlink both repos

Status: done 2026-05-09.

Work:
- Add this shared backlog file to both repos.
- Link it from Loom's open-items backlog, Loom's pairwise review, Loom's JSONL
  goal stream, and Gossamer's backlog.

Acceptance:
- `rg -n 'shared_loom_gossamer_backlog|S-LG-01' docs/backlog` finds the shared
  entry points in each repo.

### S-LG-02 - Shared backend semantics checklist

Status: implemented 2026-05-09; commit/review pending.

Work:
- Create a short review checklist used by both repos before adding UI behavior:
  backend-owned role, authority, freshness, provenance, target/source IDs,
  fixture/live distinction, and no browser-only semantic derivation.
- Keep a Gossamer-safe wording variant if the checklist is copied into the
  public-safe repo.

Implementation:
- Gossamer: `docs/backend_semantics_checklist.md` is public-safe and fixture
  focused; `npm run test:contracts` checks that the required terms remain.
- Loom: `docs/backend_semantics_checklist.md` uses live-system wording and
  keeps fixture-backed review data separate from live proof.

Acceptance:
- Checklist is referenced by Loom discovery/command-center review work and by
  Gossamer source catalogue or graph-wall review work.

### S-LG-03 - Finish Gossamer ownership tree and graph-wall manifest

Status: implemented 2026-05-09; commit/review pending.

Work:
- Complete and integrate the active Gossamer contract/UI slice for `owner_mode`, `use`,
  `format_preference`, `discovery_path`, backend-authored tree nodes, and a
  static graph-wall manifest.
- Keep every fixture fictional and generic.
- Exclude generated binaries or build artifacts from the backlog commit.

Acceptance:
- Gossamer contract tests pass.
- Source catalogue renders tree grouping from backend fixture data.
- Graph-wall manifest is served by the API and covered by fixture validation.
- Clean-room scan reports no private live-system identifiers in public fixtures,
  source code, or user-facing docs.

### S-LG-04 - Bring Gossamer fixture determinism back into Loom design fixtures

Status: open.

Work:
- Identify which Loom discovery, graph-wall, command-session, and evidence
  contracts would benefit from deterministic synthetic fixtures.
- Add fixtures only where they improve CI/review without implying live proof.

Acceptance:
- Loom validators clearly label fixture-only data and reject it for live-proof
  packets.

### S-LG-05 - Evidence report contract cross-check

Status: open.

Work:
- Compare Loom evidence-report fixtures and Gossamer campaign reports for
  shared fields: requirement reference, source provenance, anomaly summary,
  command authority, generated-at time, fixture/live status, and reviewer notes.
- Keep repo-specific terms separate.

Acceptance:
- Each repo has a small example proving how evidence remains traceable without
  pretending that simulated data validates live hardware.

### S-LG-06 - Token-optimized agent context codec benchmark

Status: in progress; Gossamer benchmark implemented 2026-05-09, Loom benchmark pending.

Work:
- Evaluate a schema-aware compact representation for repeated agent payloads:
  backlog slices, source catalogues, discovery trees, command-session event
  rings, graph manifests, and evidence reports.
- Round-trip through canonical JSON before validation, execution, persistence,
  replay, or live proof.

Acceptance:
- Measured token reduction is recorded on at least one Loom payload and one
  Gossamer payload.
- Provenance, authority, idempotency, fixture/live, and target/source identity
  fields survive compaction.

Gossamer implementation:
- `fixtures/public/agent_context_codec_benchmark.json` records public-safe
  source-catalogue and command-event-ring samples.
- `web/scripts/test-agent-context-codec.mjs` validates compact table
  round-trip behavior, required field preservation, stale metric detection, and
  a minimum measured reduction.
- `npm run test:contracts` includes the codec check.

### S-LG-07 - Discovery tree and graph assignment UX comparison

Status: implemented 2026-05-09.

Work:
- Capture the concrete Gossamer/Loom comparison and record each behavior as one of:
  shared, demo-only, or live-only.
- Use the contract as the source of truth (no inference from label text alone):
  `source_catalogue.json` for discovery tree, `source_tree_config.json` for
  operator-visible filtered views, and graph assignment from:
  `graph_wall_manifest.json` (`source_id` and `target_id`) plus
  `graph_models/*.json` (`GraphWallSignal.source`, `source_family`, and `role`).
- Shared and reusable patterns:
  - Backend-authored hierarchy (`node` -> `device` -> `subsystem` -> `stream`)
    with stream leafs resolved by `source_id`.
  - Stable source identity in table rows (`id`, `source_id`, `source_family`),
    and stable graph target IDs.
  - Dense metadata badges and fixture-owned fields for owner/use/format/provenance.
- Demo-only in Gossamer:
  - Synthetic-only `synthetic_only` campaign/manifest semantics.
  - Fixture-only assignment inputs (`fixture` source families and deterministic
    provenance).
  - Public-safe table columns and badges that are read-only.
- Intentionally live-only (absent in Gossamer):
  - Command authority/lease state, route control, and transport health controls.
  - Live readiness checks and operator execution gates.
  - Hardware routing, adapter selection, and authenticated device/session state.

Acceptance:
- Both backlogs identify which tree/assignment patterns are shared, which are
  live-only, and which are demo-only.
- Current status:
  - `docs/backlog/BACKLOG.md` records the completion note for `GOSS-10`.
  - `docs/backend_semantics_checklist.md` references this classification rule for
    future copy-on-compare work.

### S-LG-08 - Clean-room guardrail after every shared import
	
Status: implemented 2026-05-09; commit/review pending.

Work:
- Add a lightweight review gate for any Gossamer patch inspired by Loom.
- Check public fixtures, source, docs, and screenshots for private identifiers,
  real captures, real hostnames, private network addresses, protocol databases,
  and hardware-specific procedures.

Implementation:
- `docs/clean_room_import_checklist.md` defines the public-safe criteria for any
  cross-repo import.
- `web/scripts/test-clean-room.mjs` scans docs, fixtures, source, and screenshot
  artifacts for violations. It runs with `npm run test:clean-room` in `web/` and
  is included in `npm run test:contracts` at repo root.

Acceptance:
- Gossamer backlog items that reference sibling-system patterns include a
  clean-room acceptance criterion.
- The clean-room guardrail script reports no private hostnames, private lab IPs,
  credentials/secrets, protocol DB/capture artifacts, or live hardware
  procedure language in public-facing docs/fixtures/source/screenshots.

### S-LG-09 - Live vs fixture proof language audit

Status: open.

Work:
- Audit docs and backlog close-outs so "implemented", "fixture-backed",
  "smoke-tested", and "live-proven" have distinct meanings.

Acceptance:
- Loom live lanes do not close on fixture-only evidence.
- Gossamer docs do not imply hardware validation.

## Validation Commands

Loom:

```bash
jq -c . docs/backlog/goals.jsonl >/dev/null
rg -n 'shared_loom_gossamer_backlog|S-LG-' docs/backlog
git diff --check -- docs/backlog
```

Gossamer:

```bash
go test ./internal/contracts ./internal/synthetic
cd web && npm run test:contracts
rg -n 'shared_loom_gossamer_backlog|S-LG-' docs/backlog
git diff --check -- docs/backlog
```
