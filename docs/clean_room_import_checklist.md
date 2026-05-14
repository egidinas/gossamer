# Clean-Room Import Checklist (S-LG-08)

Use this list before accepting any sibling-system import into public Gossamer docs, fixtures, or source.

## Required checks

- No private hostnames or private network addresses are introduced:
  - RFC1918 / link-local private IP forms (`10.x.x.x`, `172.16-31.x.x`, `192.168.x.x`, `169.254.x.x`)
  - Hostnames containing private DNS suffixes (`.local`, `.lan`, `.internal`, `.private`, `.corp`, `.home`, etc.)
- No credentials or secret-bearing assignments appear in content:
  - `api_key`, `access token`, `secret`, `password`, `bearer token`, etc.
  - Long opaque token-like literals attached to those keys
- No protocol database or hardware protocol capture artifacts are copied into public docs/fixtures/source:
  - `*.dbc`, `*.arxml`, `*.kcd`, `*.pcap`, `*.mf4`, `*.asc`, related capture filenames, and protocol layouts
- No live hardware procedure text is imported as proof of implementation:
  - explicit commissioning/acceptance procedures, runbooks, command timing sequences, wiring-topology narratives, calibration tables
- All fixture/synthetic text remains public-safe, deterministic, and generic.
- No runtime dependency, import, package manifest, or module replacement crosses into private sibling repositories:
  - no Loom, mynaric telemetry, lab-support, jobsearch, or work-time package imports
  - no `replace github.com/egidinas/signalforge => ../...` or `/home/...` local checkout
  - no web `file:` dependency or `@loom-gossamer/shared` reintroduction

## Validation

- Run `cd web && npm run test:clean-room` after any shared-import-facing edits. This includes the information-barrier dependency/path scan.
- Keep `127.0.0.1`, loopback references for local demo runbooks, and documented legacy terms explicitly allowed in public demo files.
