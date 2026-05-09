# Gossamer

Gossamer is an environmental-test data and evidence demonstrator for exploring how requirements, test configuration, facility state, telemetry provenance, command authority, anomalies, and evidence reporting can become one repeatable operating model.

**Live hosted demo:** [gossamer.jmeyer.space](https://gossamer.jmeyer.space/)

The repository is independent, fictional, and not affiliated with any employer or customer. It does not contain real device data, real lab configuration, real protocol definitions, real facility procedures, or private identifiers.

## What It Shows

- A generic reference DUT moving through flatsat, thermal acceptance, TVac qualification, and integrated system FAT campaigns.
- Backend-owned semantic contracts for source quality, graph lanes, requirements, command authority, and evidence reports.
- A landing page, campaign graph pages, and virtual bus tap that make the demo usable as a technical portfolio walkthrough.
- A lightweight hosted static/tile deployment for reviewing the project without a local checkout.
- Deterministic synthetic fixtures that make the demo reproducible.
- A local API and operator UI that can be shown without hardware, private networks, or external services.
- A reusable portfolio artifact for discussing test-system architecture, source abstraction, and operator workflows.

## Repository Shape

- `cmd/gossamer-fixtures`: regenerates deterministic public fixtures.
- `cmd/gossamer-report`: builds evidence reports from campaign requirements and telemetry.
- `cmd/gossamer-server`: serves the local demo API.
- `internal/contracts`: backend-owned response models with `schema_version` and `generated_at`.
- `fixtures/public`: synthetic JSON and JSONL contracts served by the API and tested by the UI.
- `web`: Vite/React operator UI.
- `docs`: architecture, standards, traceability, and deployment notes.

## Agent Backlog Workflow

The canonical fixtures, reports, and backlog artifacts stay as JSON or JSONL.
For large backlog slices, source catalogues, discovery trees, evidence reports,
graph-wall fixtures, and Loom/Gossamer pairwise reviews exchanged between
agents, use the shared `@loom-gossamer/shared/agent-context-codec` package from
`/home/svc_pmg_testbed_b/shared/loom-gossamer-shared`.

The compact form is only for agent prompt/tool transport. Decode it back to
canonical JSON before changing files, regenerating fixtures, validating
contracts, publishing the public demo, or presenting evidence. The Gossamer web
contract test includes the codec consumer check:

```bash
cd web
npm run test:contracts
```

## Local Run

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

## Verification

```bash
go test ./...
cd web
npm run test:contracts
npm run test:browser
npm run build
```

`npm run test:browser` starts the local Go API and Vite app, visits every UI route at desktop and mobile widths, fails on page errors or horizontal overflow, and writes screenshots to `web/test-artifacts/screenshots/`. The artifact directory is ignored by git.

## Demo Surface

The UI exposes eight views:

- landing: project story, navigation, and compact parallel FAT preview,
- mission map: synthetic test article, facilities, buses, and campaigns,
- supervisor: swimlane board for parallel FAT and qualification activities with backend-defined hero graphs,
- graph wall: backend-defined graph lanes over deterministic telemetry,
- source catalogue: freshness, quality, and provenance for synthetic sources,
- requirement matrix: requirement results with evidence references,
- command authority: mocked lease and command path,
- bus tap: fictional data-bus virtualization view with separated TM and TC replay events,
- evidence report: campaign-level summary, anomalies, and export-ready records.

## Clean-Room Boundary

Gossamer uses fictional names, deterministic fixture data, generic DUT subsystems, generic environmental facilities, and public engineering concepts. See [docs/ip_clean_room.md](docs/ip_clean_room.md).

## Public Demo

A hosted instance is available at [https://gossamer.jmeyer.space/](https://gossamer.jmeyer.space/). It serves the lightweight public UI and precomputed telemetry tile artifacts through Cloudflare Tunnel, so the project can be reviewed from GitHub without a local checkout.

Local runs remain useful for regenerating fixtures, reports, and tile bundles. See [docs/public_demo_access.md](docs/public_demo_access.md) for the conservative deployment pattern.
