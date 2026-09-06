import type { MarketOptionsHistoryPoint, MarketOptionsHistoryWindow } from '@/libs/api';
import { OPTIONS_HISTORY_RANGE_MS, type OPTIONS_HISTORY_METRICS } from './constants';
import type { OptionsHistoryMetric, OptionsHistoryRange } from './OptionsHistory.types';

export type HistoryLinePoint = { time: number; value: number };

export function historyWindow(
  windows: MarketOptionsHistoryWindow[] | undefined,
  id: string,
): MarketOptionsHistoryWindow | undefined {
  return windows?.find((row) => row.window === id);
}

export function formatHistoryDelta(changePct: string | undefined): string {
  if (changePct == null || changePct === '') return '';
  const n = Number(changePct);
  if (!Number.isFinite(n)) return '';
  const sign = n > 0 ? '+' : '';
  return `${sign}${n.toFixed(1)}%`;
}

function pointValue(row: MarketOptionsHistoryPoint, metric: OptionsHistoryMetric): number | null {
  const raw =
    metric === 'oi' ? (row.openInterestUsd ?? row.openInterest) : metric === 'pcr' ? row.putCallOi : row.dvol;
  if (raw == null || raw === '') return null;
  const n = Number(raw);
  return Number.isFinite(n) ? n : null;
}

export function toHistoryLine(
  points: MarketOptionsHistoryPoint[] | undefined,
  metric: OptionsHistoryMetric,
  range: OptionsHistoryRange,
  nowMs = Date.now(),
): HistoryLinePoint[] {
  if (!points?.length) return [];
  const cutoff = nowMs - OPTIONS_HISTORY_RANGE_MS[range];
  const out: HistoryLinePoint[] = [];
  for (const row of points) {
    const ts = row.time ? Date.parse(row.time) : NaN;
    if (!Number.isFinite(ts) || ts < cutoff) continue;
    const value = pointValue(row, metric);
    if (value == null) continue;
    const time = Math.floor(ts / 1000);
    const prev = out[out.length - 1];
    if (prev && prev.time === time) {
      prev.value = value;
      continue;
    }
    out.push({ time, value });
  }
  return out;
}

export function historyMetricHasData(
  points: MarketOptionsHistoryPoint[] | undefined,
  metric: (typeof OPTIONS_HISTORY_METRICS)[number],
): boolean {
  return Boolean(points?.some((row) => pointValue(row, metric) != null));
}
