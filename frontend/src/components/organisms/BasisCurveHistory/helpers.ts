import type { BasisCurveMove, BasisCurvePoint, BasisCurveWindow } from '@/components/organisms/BasisPanel/BasisPanel.types';
import { BASIS_CURVE_RANGE_MS, type BASIS_CURVE_RANGES } from './constants';
import type { BasisCurveHistoryRange } from './BasisCurveHistory.types';

export type CurveLinePoint = { time: number; value: number };

export function curveWindow(
  windows: BasisCurveWindow[] | undefined,
  id: string,
): BasisCurveWindow | undefined {
  return windows?.find((row) => row.window === id);
}

export function toCurveLine(
  points: BasisCurvePoint[] | undefined,
  range: BasisCurveHistoryRange,
  nowMs = Date.now(),
): CurveLinePoint[] {
  if (!points?.length) return [];
  const cutoff = nowMs - BASIS_CURVE_RANGE_MS[range];
  const out: CurveLinePoint[] = [];
  for (const row of points) {
    const ts = row.time ? Date.parse(row.time) : NaN;
    if (!Number.isFinite(ts) || ts < cutoff) continue;
    const n = Number(row.spreadPct);
    if (!Number.isFinite(n)) continue;
    const time = Math.floor(ts / 1000);
    const prev = out[out.length - 1];
    if (prev && prev.time === time) {
      prev.value = n;
      continue;
    }
    out.push({ time, value: n });
  }
  return out;
}

export function windowChangeLabel(win: BasisCurveWindow | undefined): string {
  if (!win?.complete) return '';
  if (win.change === 'flipped' && win.pastShape && win.shape) {
    return `${win.pastShape} → ${win.shape}`;
  }
  if (win.changePct) return `${win.changePct} pp`;
  return '';
}

export function moveKindLabel(kind: BasisCurveMove['kind']): 'flipped' | 'steepened' | 'flattened' {
  if (kind === 'steepened' || kind === 'flattened') return kind;
  return 'flipped';
}

export function hasCurveHistory(
  points: BasisCurvePoint[] | undefined,
  range: (typeof BASIS_CURVE_RANGES)[number] = '24h',
  nowMs = Date.now(),
): boolean {
  return toCurveLine(points, range, nowMs).length > 1;
}
