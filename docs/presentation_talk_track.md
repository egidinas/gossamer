# Presentation Talk Track

Gossamer is a compact way to discuss system thinking without revealing protected implementation details.

## Opening

Gossamer models a generic reference DUT moving through environmental-test campaigns. The point is not a particular device. The point is the operating model: facilities, buses, sources, requirements, authority, and evidence are represented as explicit contracts rather than scattered tribal knowledge.

## Walkthrough

1. Landing page: position Gossamer as a reusable clean-room demonstrator, not a private project clone.
2. Mission map: show the fictional subsystems, facilities, buses, and campaign sequence.
3. Supervisor: show parallel FAT and qualification swimlanes, with temperature and bus-health hero graphs owned by the backend contract.
4. Source catalogue: explain why test software needs provenance, freshness, and quality before it trusts data.
5. Graph wall: show that plots are driven by backend graph contracts, not UI guesses.
6. Requirement matrix: connect synthetic telemetry to pass, fail, and inconclusive outcomes.
7. Command authority: show a mocked lease flow for controlled operations.
8. Bus tap: show fictional TM and TC replay events moving between generic nodes without exposing any real packet details.
9. Evidence report: close the loop from test execution to reviewable artifact.

## What To Emphasize

- The visible model is generic and fixture-backed by design.
- The implementation is small enough to inspect, but structured like a real product seed.
- The same pattern scales from flatsat derisking to integrated system FAT or qualification.
- Synthetic fixtures make architecture visible without needing private hardware or data.
- The bus virtualization view is a teaching model for observability and authority, not a protocol implementation.

## Natural Follow-Ups

- Replace synthetic sources with adapters in a private integration repo.
- Add authentication and audit persistence for a hosted demo.
- Add richer anomaly workflows, report exports, and campaign configuration editing.
- Use the same contracts for automated regression tests and operator training.
