import type { PumpEvent } from '@/libs/api';
import { colors, semanticColors } from '@/styles/tokens';
import { formatPumpReturnPct } from './formatPump';

/** Marker drawn on OHLCV series (Lightweight Charts series markers). */
export type ChartMarker = {
  time: number;
  position: 'aboveBar' | 'belowBar' | 'inBar';
  shape: 'arrowUp' | 'arrowDown' | 'circle' | 'square';
  color: string;
  text?: string;
  id?: string;
  size?: number;
};

/** Horizontal price line (start/end or high/low margin of a pump). */
export type ChartPriceLine = {
  id: string;
  price: number;
  color: string;
  title?: string;
  lineWidth?: number;
  /** 0 Solid, 1 Dotted, 2 Dashed, 3 LargeDashed, 4 SparseDotted */
  lineStyle?: 0 | 1 | 2 | 3 | 4;
  axisLabelVisible?: boolean;
};

const PUMP_UP = semanticColors.status.success;
const PUMP_DOWN = semanticColors.status.error;
const MARGIN_HIGH = colors.mint;
const MARGIN_LOW = colors.stone;

function eventTimeSec(openTime: string | undefined): number | null {
  if (!openTime) return null;
  const ms = Date.parse(openTime);
  if (!Number.isFinite(ms)) return null;
  return Math.floor(ms / 1000);
}

/**
 * Map pump/dump events to chart markers (arrow + return label).
 * Sorted by time ascending (required by Lightweight Charts markers).
 */
export function pumpEventsToChartMarkers(
  events: PumpEvent[] | undefined,
  options?: { max?: number },
): ChartMarker[] {
  if (!events?.length) return [];
  const max = options?.max ?? 40;
  const out: ChartMarker[] = [];
  // Prefer most recent when capping.
  const slice = events.length > max ? events.slice(-max) : events;
  for (let i = 0; i < slice.length; i++) {
    const e = slice[i];
    const time = eventTimeSec(e.openTime);
    if (time == null) continue;
    const ret = e.returnPct;
    const up = ret == null || ret >= 0;
    out.push({
      id: `pump-${e.openTime ?? i}-${e.returnPct ?? i}`,
      time,
      position: up ? 'belowBar' : 'aboveBar',
      shape: up ? 'arrowUp' : 'arrowDown',
      color: up ? PUMP_UP : PUMP_DOWN,
      text: formatPumpReturnPct(ret),
      size: 1.25,
    });
  }
  return out.sort((a, b) => a.time - b.time);
}

/**
 * Map pump events to high/low margin price lines (range of each move).
 * Caps events so the price axis stays readable.
 */
export function pumpEventsToMarginLines(
  events: PumpEvent[] | undefined,
  options?: { maxEvents?: number },
): ChartPriceLine[] {
  if (!events?.length) return [];
  const maxEvents = options?.maxEvents ?? 5;
  const slice = events.length > maxEvents ? events.slice(-maxEvents) : events;
  const lines: ChartPriceLine[] = [];
  for (let i = 0; i < slice.length; i++) {
    const e = slice[i];
    const high =
      e.high ??
      (e.endPrice != null && e.startPrice != null
        ? Math.max(e.endPrice, e.startPrice)
        : e.endPrice);
    const low =
      e.low ??
      (e.endPrice != null && e.startPrice != null
        ? Math.min(e.endPrice, e.startPrice)
        : e.startPrice);
    const retLabel = formatPumpReturnPct(e.returnPct);
    const up = e.returnPct == null || e.returnPct >= 0;
    const accent = up ? PUMP_UP : PUMP_DOWN;
    // Prefer high/low range; fall back to start/end of the measured return.
    if (high != null && Number.isFinite(high)) {
      lines.push({
        id: `margin-h-${e.openTime ?? i}`,
        price: high,
        color: MARGIN_HIGH,
        title: `${retLabel} H`,
        lineWidth: 1,
        lineStyle: 2,
        axisLabelVisible: true,
      });
    }
    if (low != null && Number.isFinite(low)) {
      lines.push({
        id: `margin-l-${e.openTime ?? i}`,
        price: low,
        color: MARGIN_LOW,
        title: `${retLabel} L`,
        lineWidth: 1,
        lineStyle: 2,
        axisLabelVisible: true,
      });
    }
    // Start / end of measured return (dotted / solid).
    if (e.startPrice != null && Number.isFinite(e.startPrice) && e.startPrice !== low && e.startPrice !== high) {
      lines.push({
        id: `margin-s-${e.openTime ?? i}`,
        price: e.startPrice,
        color: accent,
        lineWidth: 1,
        lineStyle: 1,
        axisLabelVisible: false,
      });
    }
    if (e.endPrice != null && Number.isFinite(e.endPrice) && e.endPrice !== low && e.endPrice !== high) {
      lines.push({
        id: `margin-e-${e.openTime ?? i}`,
        price: e.endPrice,
        color: accent,
        title: retLabel,
        lineWidth: 1,
        lineStyle: 0,
        axisLabelVisible: true,
      });
    }
  }
  return lines;
}
