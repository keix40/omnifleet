import Constants from "expo-constants";
import type { DispatchJob, LoginResponse } from "./types";

const gateway =
  (Constants.expoConfig?.extra?.gatewayUrl as string | undefined) ?? "http://localhost:8080";

export async function login(
  email: string,
  password: string,
  tenantSlug: string,
): Promise<LoginResponse> {
  const resp = await fetch(`${gateway}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, tenant_slug: tenantSlug }),
  });
  if (!resp.ok) {
    throw new Error("login failed");
  }
  return resp.json();
}

export async function listJobs(token: string): Promise<DispatchJob[]> {
  const resp = await fetch(`${gateway}/api/v1/dispatch/jobs`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!resp.ok) {
    throw new Error("list jobs failed");
  }
  const data = await resp.json();
  return data.jobs ?? [];
}

export async function updateJobStatus(
  token: string,
  jobId: string,
  status: string,
): Promise<void> {
  const resp = await fetch(`${gateway}/api/v1/dispatch/jobs/${jobId}/status`, {
    method: "PATCH",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ status }),
  });
  if (!resp.ok) {
    throw new Error("status update failed");
  }
}

export async function postPosition(
  token: string,
  vehicleId: string,
  latitude: number,
  longitude: number,
  speedMps: number,
  headingDeg: number,
): Promise<void> {
  const resp = await fetch(`${gateway}/api/v1/tracking/positions`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      vehicle_id: vehicleId,
      latitude,
      longitude,
      speed_mps: speedMps,
      heading_deg: headingDeg,
    }),
  });
  if (!resp.ok) {
    throw new Error("ingest failed");
  }
}
