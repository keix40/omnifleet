import type { QueuedPosition } from "./types";

const MAX_QUEUE = 200;

export class LocationQueue {
  private queue: QueuedPosition[] = [];

  enqueue(point: QueuedPosition): void {
    this.queue.push(point);
    if (this.queue.length > MAX_QUEUE) {
      this.queue.shift();
    }
  }

  drainBatch(max: number): QueuedPosition[] {
    const batch = this.queue.splice(0, max);
    return batch;
  }

  size(): number {
    return this.queue.length;
  }

  shouldFlush(lastFlushMs: number, nowMs: number, minIntervalMs: number): boolean {
    return nowMs-lastFlushMs >= minIntervalMs && this.queue.length > 0;
  }
}

export function throttlePositions(
  points: QueuedPosition[],
  minIntervalMs: number,
): QueuedPosition[] {
  if (points.length === 0) {
    return points;
  }
  const out: QueuedPosition[] = [points[0]];
  for (let i = 1; i < points.length; i++) {
    if (points[i].recorded_at-out[out.length - 1].recorded_at >= minIntervalMs) {
      out.push(points[i]);
    }
  }
  return out;
}
