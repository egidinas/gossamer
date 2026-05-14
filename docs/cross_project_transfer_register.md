# Cross-Project Transfer Register

This register keeps strong reusable ideas from getting trapped in one repo while
preserving the public information barrier. It is not a dependency map. It is a
review checklist for deciding what should become a public SignalForge primitive,
what belongs only in Gossamer's deterministic demo, and what must stay in a
private downstream implementation.

## Transfer Rules

- Promote reusable contracts to SignalForge, then consume them from Gossamer.
- Keep Gossamer public-safe: deterministic fixtures, generic names, no private
  host paths, captures, credentials, serial numbers, or hardware procedures.
- Treat sibling-system behavior as inspiration only until it is restated as a
  neutral contract with synthetic fixtures.
- Record route/source provenance explicitly so UI and backend behavior can show
  where a value came from without exposing private infrastructure.

## Candidate Patterns

| Pattern | Public home | Implementation detail to preserve | Do not carry over |
| --- | --- | --- | --- |
| Backend-owned semantics | SignalForge contract, Gossamer fixture/API | Backend classifies role, authority, freshness, provenance, and display grouping before the browser renders it. | Frontend inference based on labels, private device names, or transport-specific strings. |
| Signal identity and provenance | SignalForge | Stable signal IDs with type, subtype, parameter, device, instance, source route, and readable alias fields. | Real serial numbers, private aliases, lab hostnames, or raw private catalogues. |
| Multi-route acquisition | SignalForge contract; private repos implement adapters | Preferred route plus fallback routes are equivalent to the data backend and UI, with route status visible. | Hard dependency on one private transport stack. |
| Ring-buffer layering | SignalForge contract; Gossamer fixture model | Device, edge RAM, edge flash, and backend history can be merged by sample identity and source route without duplicates. | Real ring files, raw captures, or flash-layout assumptions. |
| Polling scheduler | SignalForge primitive | Priority queue, manual-front insertion, high-priority continuous reads, low-priority round-robin, and graceful degradation under congestion. | Controller-specific command details outside the owning adapter package. |
| Batch read framing | SignalForge primitive when generic; adapter-specific framing private | Batch/multi-parameter reads should report partial success, timing, and per-value provenance. | Protocol frames that are only valid for one private device family unless published by that adapter. |
| Reduction pipeline | SignalForge | Polling stays fast; reduction is asynchronous, SNR-aware, and consumer-rate driven. | UI-side downsampling that hides acquisition gaps or changes authority semantics. |
| Transition characterization | SignalForge analytics contract; Gossamer demo fixture | Observe every state transition, capture response windows, and produce candidate tuning evidence without automatic authority escalation. | Unreviewed automatic control writes or private plant models. |
| PID advisor | SignalForge analytics contract; private repos implement write policy | Separate observation, recommendation, approved device self-tune, and minor bounded adjustments. | Autonomous tuning without explicit authority, rollback, and audit trail. |
| Thermal power model | SignalForge physical model contract; adapter-specific calibration private | Channel mode controls which model applies; aggregate heat, electrical power, and pumped heat are separate outputs. | Device-specific calibration constants or non-public datasheets. |
| Graph wall provenance | Gossamer UI contract; SignalForge signal metadata | Every plotted series shows source route, device/instance identity, fixture/live status, and history depth. | Visuals that imply live hardware proof in public fixtures. |

## Current Public-Safe Carryover Set

- Keep Gossamer focused on deterministic graph-wall, source-tree, provenance,
  authority, and evidence-report fixtures.
- Keep SignalForge as the only reusable code dependency for shared primitives.
- Keep device-family adapters, live route ownership, private catalogues, and
  hardware procedures outside this public repo unless they are independently
  published as public-safe packages.

## Review Checklist

Before adopting a cross-project idea here:

1. State the idea as a neutral contract or fixture behavior.
2. Confirm whether it belongs in SignalForge or Gossamer-only demo code.
3. Add synthetic fixture coverage before UI rendering depends on it.
4. Run `npm run test:clean-room`.
5. Reject the change if it imports, references, or reconstructs private project
   internals.
