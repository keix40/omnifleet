# ADR 0004: Geofencing via NATS consumer (not tracking gRPC)

## Status

Accepted

## Context

Tracking publishes GPS positions to JetStream. Geofencing must evaluate PostGIS boundaries, persist enter/exit state, and publish alert events for the gateway WebSocket fan-out.

Two integration styles were possible:

1. **Synchronous:** tracking calls geofencing `EvaluatePosition` over gRPC on every ingest.
2. **Event-driven:** geofencing runs a durable JetStream consumer on `fleet.<tenant_id>.positions`.

Using both would duplicate evaluation and could emit duplicate alerts.

## Decision

**Production geofencing runs only on the NATS JetStream consumer** inside the geofencing service. Tracking **does not** call geofencing gRPC.

The geofencing gRPC `EvaluatePosition` RPC remains for health checks, manual debugging, and future admin tools — not for the hot ingest path.

## Rationale

- Matches the vertical-slice event flow (ADR 0001): ingest → stream → side effects → alerts stream → gateway.
- Decouples tracking latency from PostGIS evaluation and allows independent scaling/restarts of geofencing.
- E2E compose tests prove the full pipeline (tracking → NATS → geofencing → alerts → gateway WS) without in-process shortcuts.

## Consequences

- Geofencing must keep its consumer alive for the process lifetime (hold `ConsumeContext`, stop on shutdown).
- Alert publishes happen after the tenant transaction commits so WebSocket clients never see events for rolled-back state.
- Compose and Kubernetes deploy geofencing with `NATS_URL`; tracking does not need `GEOFENCING_GRPC_ADDR`.
- Geofence evaluation must finish reading the geofence `rows` cursor before running follow-up queries on the same pgx transaction (otherwise pgx returns `conn busy` and alerts never publish).
