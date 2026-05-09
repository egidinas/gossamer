# Repository Guidelines

## Project Shape

Gossamer is the public-safe demonstrator for deterministic environmental-test
evidence, source catalogues, graph walls, command authority, and operator UI
contracts. Keep it independent from Loom live hardware and private lab details.

Start with:

- `README.md`
- `docs/ip_clean_room.md`
- `docs/public_demo_access.md`
- `/home/svc_pmg_testbed_b/.codex/skills/gossamer/SKILL.md`
- `/home/svc_pmg_testbed_b/shared/loom-gossamer-shared/docs/backlog/shared_loom_gossamer_backlog.md`

## Core Invariants

- Backend contracts and deterministic fixtures own semantics. The browser
  renders source quality, graph lanes, command authority, evidence status,
  freshness, units, and provenance.
- Keep the clean-room boundary intact: fictional devices, generic test
  campaigns, synthetic data, no private protocol databases, no real captures,
  no lab node names, no serial numbers, and no hardware procedures.
- Model rich decoded telemetry as the default. Raw or low-level bus views are
  compatibility demonstrations, not the primary operator contract.
- Enum and boolean labels require a durable dictionary contract. Signal kind is
  only a hint; late-joining views must be able to resolve labels from fixtures,
  catalogues, sidecar metadata, or dictionary events without replaying old
  samples.
- Shared Loom/Gossamer planning belongs in the shared repo backlog. Do not
  create divergent local copies of shared S-LG items.

## Agent Context Efficiency

Canonical fixtures, reports, contracts, and backlog files remain JSON or JSONL.
For large backlog slices, source catalogues, discovery trees, evidence reports,
graph-wall fixtures, pairwise Loom/Gossamer reviews, or other repeated
agent-facing JSON payloads, use `@loom-gossamer/shared/agent-context-codec`
from `/home/svc_pmg_testbed_b/shared/loom-gossamer-shared`.

Decode compact agent-context payloads back to canonical JSON before editing
files, regenerating fixtures, validating contracts, publishing, replaying, or
presenting evidence.

## Verification

Use narrow checks for the touched lane:

```bash
go test ./...
cd web && npm run test:contracts
cd web && npm run build
git diff --check
```
