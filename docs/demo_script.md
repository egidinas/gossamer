# Demo Script

This script is for a short public portfolio walkthrough.

## Setup

```bash
go run ./cmd/gossamer-fixtures
go run ./cmd/gossamer-report --campaign flatsat_derisking
go run ./cmd/gossamer-report --campaign thermal_acceptance_fat
go run ./cmd/gossamer-report --campaign tvac_qualification
go run ./cmd/gossamer-report --campaign integrated_system_fat
go run ./cmd/gossamer-server
```

In another shell:

```bash
cd web
npm install
npm run dev -- --host 127.0.0.1 --port 5179
```

Open `http://127.0.0.1:5179/#landing`.

## Walkthrough

1. Start on the landing page and frame Gossamer as a clean-room demonstrator for environmental-test system thinking.
2. Open Mission Map and show the fictional `AuroraSat-1` article, generic facilities, and campaign sequence.
3. Open Supervisor and use the swimlanes to explain parallel FAT activity: thermal ramp, EPS load step, command script, RF simulator, payload heater cycling, archive capture, and interlock monitoring.
4. Open Graph Wall and point out that graph lanes are backend-defined contracts.
5. Open Sources and show freshness, provenance, and quality as first-class operating data.
6. Open Requirements and connect telemetry to pass, warning, and evidence references.
7. Open Commands and show the fictional command-authority lease path.
8. Open Bus Tap and explain TM/TC visibility as a public-safe virtualization mockup, not a real protocol.
9. Open Evidence and close with repeatable report generation across flatsat, FAT, and qualification campaigns.

## Closing Point

The artifact is intentionally small, but it demonstrates the operating model: contracts first, deterministic evidence, clean boundaries, and operator views that can scale into a private implementation.
