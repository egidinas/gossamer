# Public Variant Readiness

Date: 2026-05-12

Gossamer is the public fictional SignalForge demo. It must remain useful without
hardware, private networks, real captures, real protocol databases, or private
procedures.

## Gates

- Build and test from a fresh clone using public dependencies only.
- Depend on public SignalForge tags only after SignalForge exists; do not depend
  on the internal shared staging repository in public branches.
- Keep all fixtures fictional and deterministic.
- Preserve backend-owned semantics: the browser renders role, authority,
  freshness, provenance, and source/target identity from fixtures or APIs.
- Clean-room scan `fixtures/public`, `internal`, `web/src`, and public docs
  before every public release.

## Local Verification Set

- `go run ./cmd/gossamer-fixtures`
- `go test ./...`
- `cd web && npm run test:contracts`
- `cd web && npm run build`
- `cd web && npm run test:browser`

## Allowed Imports From Legacy Work

- Generic graph/tile interaction patterns.
- Fictional archive/replay and requirement-evidence examples.
- Public-safe source catalogue and command authority concepts.

## Rejected Imports

- Live adapter names, private bus labels, lab topology, screenshots, captures,
  serials, DBCs, customer procedures, or operational proof claims.
