# Backend Semantics Checklist

Use this checklist before merging source catalogue, graph wall, command
authority, bus tap, evidence report, or agent-context changes. Gossamer is a
public-safe deterministic demonstrator, so every item must remain generic and
fixture-owned.

## Required Review Points

- backend-owned role: role, grouping, units, graph target, and evidence meaning
  come from fixtures or API contracts, not from browser-side name parsing.
- authority: command or control affordances expose the fixture authority model,
  lease state, and mock status before rendering actions.
- freshness: timestamps, generated-at values, fixture version, and replay window
  are visible in the backend contract when freshness affects interpretation.
- provenance: source, report, graph, and generated data carry enough provenance
  to trace which deterministic fixture or generator produced them.
- source and target identity: source IDs, target IDs, report references, and
  graph assignment IDs are stable explicit fields, not derived from display
  labels.
- fixture/live distinction: public Gossamer data is fixture-only and must not
  be described as hardware validation, live proof, or operational telemetry.
- no browser-only semantic derivation: the UI may sort, collapse, filter, and
  display backend-authored semantics, but it must not invent authority, safety,
  evidence, live status, subsystem, or graph meaning locally.
- parity imports from sibling systems: if a source-tree or graph-assignment UX
  pattern is adopted, backlog and review notes must state whether it is shared,
  demo-only, or intentionally live-only.

## Review Use

Reference this checklist from backlog close-outs or review notes whenever a
change touches the source catalogue, graph wall, bus tap, command authority, or
evidence report. A change fails review if a user-facing behavior depends on a
semantic guess that is absent from the fixture or API contract.

For shared-import gatekeeping, also run the clean-room import guardrail in
`docs/clean_room_import_checklist.md` and `web/scripts/test-clean-room.mjs`.
