export type LoginResponse = {
  access_token: string;
  tenant_id: string;
  user_id: string;
};

export type DispatchJob = {
  id: string;
  status: string;
  pickup_label?: string;
  dropoff_label?: string;
  vehicle_id?: string;
};

export type QueuedPosition = {
  vehicle_id: string;
  latitude: number;
  longitude: number;
  speed_mps: number;
  heading_deg: number;
  recorded_at: number;
};
