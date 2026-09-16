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

function pointValue(row: MarketOptionsHistoryPoint, metric: OptionsHistoryMetric | 'index'): number | null {
  const raw =
    metric === 'oi'
      ? (row.openInterestUsd ?? row.openInterest)
      : metric === 'pcr'
        ? row.putCallOi
        : metric === 'gex'
          ? row.gammaNetUsd
          : metric === 'flip'
            ? row.gammaFlip
            : metric === 'index'
              ? row.indexPrice
              : row.dvol;
  if (raw == null || raw === '') return null;
  const n = Number(raw);
  return Number.isFinite(n) ? n : null;
}

export function toHistoryLine(
  points: MarketOptionsHistoryPoint[] | undefined,
  metric: OptionsHistoryMetric | 'index',
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

export function windowChipLabel(
  win: MarketOptionsHistoryWindow | undefined,
  metric: OptionsHistoryMetric,
): { delta: string; note?: 'regime' | 'flipSide'; to?: string } {
  if (metric === 'gex') {
    if (win?.regime && win.regimeWas && win.regime !== win.regimeWas) {
      return { delta: win.regime, note: 'regime', to: win.regime };
    }
    return { delta: formatHistoryDelta(win?.gamma?.changePct) };
  }
  if (metric === 'flip') {
    if (win?.flipSide && win.flipSideWas && win.flipSide !== win.flipSideWas) {
      return { delta: win.flipSide, note: 'flipSide', to: win.flipSide };
    }
    return { delta: formatHistoryDelta(win?.flip?.changePct) };
  }
  const delta =
    metric === 'oi'
      ? formatHistoryDelta(win?.oi?.changePct)
      : metric === 'pcr'
        ? formatHistoryDelta(win?.putCall?.changePct)
        : formatHistoryDelta(win?.dvol?.changePct);
  return { delta };
}
