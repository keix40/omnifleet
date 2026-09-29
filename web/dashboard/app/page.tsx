"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import Map, { Marker } from "react-map-gl/maplibre";
import "maplibre-gl/dist/maplibre-gl.css";
import { PlatformPanels } from "./PlatformPanels";

const gateway = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080";
const wsBase = process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080";

type PositionEvent = {
  tenant_id: string;
  vehicle_id: string;
  latitude: number;
  longitude: number;
  speed_mps?: number;
};

type AlertEvent = {
  tenant_id: string;
  vehicle_id: string;
  geofence_name?: string;
  event_type?: string;
  message?: string;
};

export default function HomePage() {
  const [token, setToken] = useState<string | null>(null);
  const [email, setEmail] = useState("dispatcher@acme.test");
  const [password, setPassword] = useState("demo-password-change-me");
  const [tenantSlug, setTenantSlug] = useState("acme-logistics");
  const [error, setError] = useState<string | null>(null);
  const [position, setPosition] = useState<PositionEvent | null>(null);
  const [alerts, setAlerts] = useState<AlertEvent[]>([]);

  const onLogin = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    const resp = await fetch(`${gateway}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password, tenant_slug: tenantSlug }),
    });
    if (!resp.ok) {
      setError("Login failed");
      return;
    }
    const data = await resp.json();
    setToken(data.access_token ?? data.accessToken);
  };

  useEffect(() => {
    if (!token) return;
    let ws: WebSocket | null = null;
    let cancelled = false;

    (async () => {
      const ticketResp = await fetch(`${gateway}/api/v1/ws/fleet/ticket`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!ticketResp.ok || cancelled) return;
      const ticketData = await ticketResp.json();
      const subprotocol = `omnifleet.v1.${ticketData.ticket}`;
      ws = new WebSocket(`${wsBase}/api/v1/ws/fleet/live`, [subprotocol]);
      ws.onmessage = (msg) => {
        try {
          const payload = JSON.parse(msg.data);
          if (payload.latitude !== undefined) setPosition(payload as PositionEvent);
          if (payload.event_type) setAlerts((p) => [payload as AlertEvent, ...p].slice(0, 8));
        } catch {
          /* ignore */
        }
      };
    })();

    return () => {
      cancelled = true;
      ws?.close();
    };
  }, [token]);

  const viewState = useMemo(
    () => ({
      longitude: position?.longitude ?? -122.4194,
      latitude: position?.latitude ?? 37.7749,
      zoom: 12,
    }),
    [position]
  );

  if (!token) {
    return (
      <main style={{ maxWidth: 420, margin: "4rem auto", padding: "0 1rem" }}>
        <h1>OmniFleet Dispatcher</h1>
        <p>Sign in with a seeded demo tenant user (Acme or Globex).</p>
        <form className="card" onSubmit={onLogin}>
          <label>
            Email
            <input value={email} onChange={(e) => setEmail(e.target.value)} style={{ width: "100%" }} />
          </label>
          <label>
            Tenant slug
            <input
              value={tenantSlug}
              onChange={(e) => setTenantSlug(e.target.value)}
              style={{ width: "100%" }}
            />
          </label>
          <label>
            Password
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              style={{ width: "100%" }}
            />
          </label>
          {error && <p style={{ color: "#ff8e8e" }}>{error}</p>}
          <button type="submit">Sign in</button>
        </form>
      </main>
    );
  }

  return (
    <main style={{ display: "grid", gridTemplateRows: "auto 1fr", height: "100vh" }}>
      <header style={{ padding: "0.75rem 1rem", borderBottom: "1px solid #24304d" }}>
        <strong>Live fleet map</strong> <span style={{ opacity: 0.7 }}>(tenant-scoped WebSocket feed)</span>
      </header>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 320px", minHeight: 0 }}>
        <Map
          {...viewState}
          onMove={(evt) => {
            /* keep controlled by latest position */
            void evt;
          }}
          mapStyle="https://demotiles.maplibre.org/style.json"
          style={{ width: "100%", height: "100%" }}
        >
          {position && (
            <Marker longitude={position.longitude} latitude={position.latitude} anchor="center">
              <div
                style={{
                  width: 14,
                  height: 14,
                  borderRadius: "50%",
                  background: "#4ade80",
                  border: "2px solid white",
                }}
              />
            </Marker>
          )}
        </Map>
        <aside className="card" style={{ overflow: "auto", margin: "0.75rem" }}>
          <h3>Alerts</h3>
          {alerts.length === 0 && <p>No geofence events yet. Start the GPS simulator.</p>}
          <ul>
            {alerts.map((a, i) => (
              <li key={i}>
                {a.event_type} — {a.geofence_name ?? a.message}
              </li>
            ))}
          </ul>
          <PlatformPanels token={token} />
        </aside>
      </div>
    </main>
  );
}
