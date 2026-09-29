import { LocationQueue, throttlePositions } from "../src/locationQueue";

describe("LocationQueue", () => {
  it("drains batches in order", () => {
    const q = new LocationQueue();
    q.enqueue({
      vehicle_id: "v1",
      latitude: 1,
      longitude: 2,
      speed_mps: 3,
      heading_deg: 4,
      recorded_at: 1000,
    });
    const batch = q.drainBatch(10);
    expect(batch).toHaveLength(1);
    expect(q.size()).toBe(0);
  });

  it("throttles by interval", () => {
    const points = [
      { vehicle_id: "v", latitude: 0, longitude: 0, speed_mps: 1, heading_deg: 0, recorded_at: 0 },
      { vehicle_id: "v", latitude: 0, longitude: 0, speed_mps: 1, heading_deg: 0, recorded_at: 500 },
      { vehicle_id: "v", latitude: 0, longitude: 0, speed_mps: 1, heading_deg: 0, recorded_at: 3000 },
    ];
    const out = throttlePositions(points, 2000);
    expect(out).toHaveLength(2);
  });
});
