"use client";

import { useCallback, useEffect, useState } from "react";

const gateway = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080";

type Props = { token: string };

export function PlatformPanels({ token }: Props) {
  const [subscription, setSubscription] = useState<Record<string, unknown> | null>(null);
  const [usage, setUsage] = useState<Record<string, unknown> | null>(null);
  const [prefs, setPrefs] = useState<Record<string, unknown> | null>(null);
  const [jobs, setJobs] = useState<unknown[]>([]);
  const [eta, setEta] = useState<Record<string, unknown> | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const headers = {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
  };

  const load = useCallback(async () => {
    const [sub, use, pref, jobList] = await Promise.all([
      fetch(`${gateway}/api/v1/billing/subscription`, { headers }).then((r) => r.json()),
      fetch(`${gateway}/api/v1/billing/usage`, { headers }).then((r) => r.json()),
      fetch(`${gateway}/api/v1/notifications/preferences`, { headers }).then((r) => r.json()),
      fetch(`${gateway}/api/v1/dispatch/jobs`, { headers }).then((r) => r.json()),
    ]);
    setSubscription(sub);
    setUsage(use);
    setPrefs(pref);
    setJobs(jobList.jobs ?? []);
  }, [token]);

  useEffect(() => {
    load().catch(() => setMessage("Failed to load platform data"));
  }, [load]);

  const createJob = async () => {
    const resp = await fetch(`${gateway}/api/v1/dispatch/jobs`, {
      method: "POST",
      headers,
      body: JSON.stringify({
        pickup: { latitude: 37.772, longitude: -122.425 },
        dropoff: { latitude: 37.785, longitude: -122.41 },
        pickup_label: "Dashboard pickup",
        dropoff_label: "Dashboard dropoff",
        auto_assign: true,
      }),
    });
    if (!resp.ok) {
      setMessage("Create job failed");
      return;
    }
    await load();
  };

  const computeEta = async () => {
    const resp = await fetch(`${gateway}/api/v1/eta/compute`, {
      method: "POST",
      headers,
      body: JSON.stringify({
        vehicle_id: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
        publish_update: true,
        destination: { latitude: 37.785, longitude: -122.41 },
      }),
    });
    setEta(await resp.json());
  };

  return (
    <div style={{ marginTop: "1rem" }}>
      <h3>Dispatch & ETA</h3>
      <button type="button" onClick={createJob}>
        Create + auto-assign job
      </button>{" "}
      <button type="button" onClick={computeEta}>
        Compute ETA
      </button>
      {eta && (
        <pre style={{ fontSize: 11 }}>{JSON.stringify(eta, null, 2)}</pre>
      )}
      <ul>
        {jobs.slice(0, 5).map((j: any) => (
          <li key={j.id}>
            {j.id?.slice(0, 8)} — {String(j.status)}
          </li>
        ))}
      </ul>
      <h3>Billing</h3>
      <pre style={{ fontSize: 11 }}>{JSON.stringify({ subscription, usage }, null, 2)}</pre>
      <h3>Notifications</h3>
      <pre style={{ fontSize: 11 }}>{JSON.stringify(prefs, null, 2)}</pre>
      {message && <p style={{ color: "#ff8e8e" }}>{message}</p>}
    </div>
  );
}
