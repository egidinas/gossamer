# Gossamer

Gossamer is a public-safe synthetic spacecraft environmental-test demonstrator for explaining how requirements, test configuration, facility state, telemetry provenance, command authority, anomalies, and evidence reporting can become one repeatable operating model.

The repository is independent, fictional, and not affiliated with any employer or customer. It does not contain real spacecraft data, real lab configuration, real protocol definitions, real facility procedures, or private identifiers.

## What It Shows

- A fictional `AuroraSat-1` test article moving through flatsat, thermal acceptance, TVAC qualification, and integrated system FAT campaigns.
- Backend-owned semantic contracts for source quality, graph lanes, requirements, command authority, and evidence reports.
- Deterministic synthetic fixtures that make the demo reproducible.
- A local API and operator UI that can be shown without hardware, private networks, or external services.

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

## Clean-Room Boundary

Gossamer uses only fictional names, deterministic synthetic data, generic spacecraft subsystems, generic environmental facilities, and public engineering concepts. See [docs/ip_clean_room.md](docs/ip_clean_room.md).

