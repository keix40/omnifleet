# ADR 0003: REST/JSON gateway with WebSocket fan-out

## Status

Accepted

## Context

Clients include browsers, mobile drivers, and future partner integrations. Internal services speak gRPC.

## Decision

Expose **REST/JSON** and **WebSocket** on the gateway; keep gRPC east-west between services.

## Rationale

- Browsers and Expo clients integrate easily with HTTPS JSON.
- WebSockets deliver live map updates without polling.
- gRPC preserves typed contracts code-generated from `proto/`.

## Consequences

- Gateway validates JWT locally (shared secret with auth service in dev; mTLS + JWKS later).
- Browser WebSocket uses `/api/v1/ws/fleet/live?access_token=` query parameter (header-less fallback).
