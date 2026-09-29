# ADR 0001: NATS JetStream as the event bus

## Status

Accepted

## Context

OmniFleet needs durable, tenant-scoped event streams for GPS positions, geofence alerts, and future dispatch/billing side effects. Kafka and NATS JetStream were candidates.

## Decision

Use **NATS JetStream** with subject hierarchy `fleet.<tenant_id>.{positions|alerts}`.

## Rationale

- **Operational surface**: single lightweight binary, simpler local docker-compose story than ZooKeeper/KRaft Kafka.
- **Multi-tenant isolation**: subject prefixes map cleanly to tenant IDs and gateway consumers.
- **Latency**: sub-millisecond fan-out suits live dispatcher maps.
- **Durability**: JetStream file storage meets GPS replay requirements for the vertical slice; Kafka remains an option if ordering/partition throughput demands grow.

## Consequences

- Services use `github.com/nats-io/nats.go` JetStream APIs.
- CI and compose run `nats -js`.
- Future cross-region DR may require JetStream super-cluster or reassessment for Kafka.
