# Bus Virtualization Mockup

The Gossamer bus tap is a fictional teaching model for observing data movement between generic nodes. It is not a real bus implementation, packet decoder, command dictionary, or transport adapter.

## Model

The fixture `fixtures/public/bus_virtualization_tap.json` describes one synthetic connection:

- source and destination nodes such as `flatsat_rack_a`, `aurorasat_1`, `archive_node_a`, `thermal_chamber_a`, and `tvac_chamber_q1`,
- generic buses such as `telemetry_bus`, `command_bus`, and `facility_control_bus`,
- stream health fields including latency, freshness, packet counters, dropped-frame count, and quality state,
- recent replay events with fictional envelope IDs such as `BUS-TM-0001` and `BUS-TC-0001`.

## TM And TC Separation

`TM` means a generic telemetry event flowing from a source toward the observer or archive. `TC` means a generic command request and acknowledgement path. The distinction is useful for explaining authority, auditability, source freshness, and live observability.

The model deliberately avoids:

- binary payloads,
- message IDs,
- field offsets,
- command opcodes,
- bus timing rules,
- private vocabulary,
- real source names or node addresses.

## API

`GET /api/bus-tap` returns a deterministic top-level contract with `schema_version`, `generated_at`, stream metadata, and recent events. The current implementation is polling-friendly: refreshing the endpoint gives a stable replay window suitable for a local demo. A private downstream system could replace this with a live adapter, but that adapter should live outside this public repository.

## UI

The Bus Tap page renders bidirectional TM and TC columns, stream cards, counters, latency, source and destination nodes, and recent event summaries. It is designed to look like a live operational tap while remaining entirely synthetic.
