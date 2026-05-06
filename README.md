# Gossamer

Gossamer is a public-safe synthetic spacecraft environmental-test demonstrator for explaining how requirements, test configuration, facility state, telemetry provenance, command authority, anomalies, and evidence reporting can become one repeatable operating model.

The repository is independent, fictional, and not affiliated with any employer or customer. It does not contain real spacecraft data, real lab configuration, real protocol definitions, real facility procedures, or private identifiers.

## What It Shows

- A fictional `AuroraSat-1` test article moving through flatsat, thermal acceptance, TVAC qualification, and integrated system FAT campaigns.
- Backend-owned semantic contracts for source quality, graph lanes, requirements, command authority, and evidence reports.
- Deterministic synthetic fixtures that make the demo reproducible.
- A local API and operator UI that can be shown without hardware, private networks, or external services.
- A reusable portfolio artifact for discussing test-system architecture, clean-room abstraction, and operator workflows without exposing protected work.

## Repository Shape

- `cmd/gossamer-fixtures`: regenerates deterministic public fixtures.
- `cmd/gossamer-report`: builds evidence reports from campaign requirements and telemetry.
- `cmd/gossamer-server`: serves the local demo API.
- `internal/contracts`: backend-owned response models with `schema_version` and `generated_at`.
- `fixtures/public`: synthetic JSON and JSONL contracts served by the API and tested by the UI.
- `web`: Vite/React operator UI.
- `docs`: clean-room, architecture, standards, traceability, and public-demo notes.

## Local Run

```bash
go run ./cmd/gossamer-fixtures
go run ./cmd/gossamer-report --campaign thermal_acceptance_fat
go run ./cmd/gossamer-report --campaign tvac_qualification
go run ./cmd/gossamer-server
```

In another shell:

```bash
cd web
npm install
npm run dev -- --host 127.0.0.1 --port 5179
```

Open `http://127.0.0.1:5179/#mission-map`.

## Verification

```bash
go test ./...
cd web
npm run test:contracts
npm run build
```

## Demo Surface

The UI exposes six views:

- mission map: synthetic test article, facilities, buses, and campaigns,
- graph wall: backend-defined graph lanes over deterministic telemetry,
- source catalogue: freshness, quality, and provenance for synthetic sources,
- requirement matrix: requirement results with evidence references,
- command authority: mocked lease and command path,
- evidence report: campaign-level summary, anomalies, and export-ready records.

## Clean-Room Boundary

Gossamer uses only fictional names, deterministic synthetic data, generic spacecraft subsystems, generic environmental facilities, and public engineering concepts. See [docs/ip_clean_room.md](docs/ip_clean_room.md).

## Public Demo

The project is intended to run locally first. For temporary public access, serve the Go API behind a narrow reverse proxy and serve the built web assets as static files. See [docs/public_demo_access.md](docs/public_demo_access.md) for a conservative deployment pattern.
