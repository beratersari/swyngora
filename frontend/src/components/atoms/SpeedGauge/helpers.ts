import { GAUGE_CX, GAUGE_CY, GAUGE_R } from './constants';

export function clampGauge(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.min(100, Math.max(0, value));
}

/** Point on the upper semicircle. 0 is the left end, 100 is the right end. */
export function gaugePoint(value: number, r = GAUGE_R, cx = GAUGE_CX, cy = GAUGE_CY) {
  const theta = Math.PI * (1 - clampGauge(value) / 100);
  return {
    x: roundGauge(cx + r * Math.cos(theta)),
    y: roundGauge(cy - r * Math.sin(theta)),
  };
}

/** Upper arc from `from` to `to` (both 0–100). */
export function gaugeArc(from: number, to: number, r = GAUGE_R): string {
  const start = gaugePoint(from, r);
  const end = gaugePoint(to, r);
  return `M ${start.x} ${start.y} A ${r} ${r} 0 0 0 ${end.x} ${end.y}`;
}

function roundGauge(n: number): number {
  return Math.round(n * 10) / 10;
}
