# Presentation Talk Track

Gossamer is a compact way to discuss system thinking without revealing protected implementation details.

## Opening

Gossamer models a fictional spacecraft test program called `AuroraSat-1`. The point is not the spacecraft. The point is the operating model: facilities, buses, sources, requirements, authority, and evidence are represented as explicit contracts rather than scattered tribal knowledge.

## Walkthrough

1. Mission map: show the fictional subsystems, facilities, buses, and campaign sequence.
2. Source catalogue: explain why test software needs provenance, freshness, and quality before it trusts data.
3. Graph wall: show that plots are driven by backend graph contracts, not UI guesses.
4. Requirement matrix: connect synthetic telemetry to pass, fail, and inconclusive outcomes.
5. Command authority: show a mocked lease flow for controlled operations.
6. Evidence report: close the loop from test execution to reviewable artifact.

## What To Emphasize

- This is clean-room and public-safe by design.
- The implementation is small enough to inspect, but structured like a real product seed.
- The same pattern scales from flatsat derisking to integrated system FAT or qualification.
- Synthetic fixtures make architecture visible without needing private hardware or data.

## Natural Follow-Ups

- Replace synthetic sources with adapters in a private integration repo.
- Add authentication and audit persistence for a hosted demo.
- Add richer anomaly workflows, report exports, and campaign configuration editing.
- Use the same contracts for automated regression tests and operator training.
