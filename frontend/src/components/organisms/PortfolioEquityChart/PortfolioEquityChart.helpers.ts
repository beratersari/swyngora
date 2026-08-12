import type { PortfolioEquityPoint, PortfolioPerformancePeriod } from '@/libs/api';
import { TickMarkType, type Time } from 'lightweight-charts';

export type EquityLinePoint = { time: Time; value: number };

/** Map API equity samples to chart series; guarantee ≥2 points when possible for a visible line. */
export function toEquityLineData(
  points: PortfolioEquityPoint[],
  opts?: { startEquity?: number; startAt?: string },
): EquityLinePoint[] {
  const data = points
    .map((p) => {
      const ts = p.t ? Math.floor(Date.parse(p.t) / 1000) : NaN;
      const eq = p.equity;
      if (!Number.isFinite(ts) || eq == null || !Number.isFinite(eq)) return null;
      return { time: ts as Time, value: eq };
    })
    .filter((x): x is EquityLinePoint => x != null)
    .sort((a, b) => Number(a.time) - Number(b.time));

  // Drop exact duplicate timestamps (LWC rejects them).
  const deduped: EquityLinePoint[] = [];
  for (const p of data) {
    const prev = deduped[deduped.length - 1];
    if (prev && Number(prev.time) === Number(p.time)) {
      deduped[deduped.length - 1] = p;
      continue;
    }
    deduped.push(p);
  }

  if (deduped.length >= 2) return deduped;
  if (deduped.length === 1) {
    const only = deduped[0];
    const startTs = opts?.startAt ? Math.floor(Date.parse(opts.startAt) / 1000) : NaN;
    const startEq =
      opts?.startEquity != null && Number.isFinite(opts.startEquity) ? opts.startEquity : only.value;
    if (Number.isFinite(startTs) && startTs < Number(only.time)) {
      return [{ time: startTs as Time, value: startEq }, only];
    }
    // Same-second fallback: one minute earlier so LWC can draw a segment.
    const earlier = (Number(only.time) - 60) as Time;
    return [{ time: earlier, value: startEq }, only];
  }
  return deduped;
}

export function equitySpanSeconds(data: EquityLinePoint[]): number {
  if (data.length < 2) return 0;
  return Math.max(0, Number(data[data.length - 1].time) - Number(data[0].time));
}

function timeToDate(time: Time): Date {
  if (typeof time === 'number') return new Date(time * 1000);
  if (typeof time === 'string') return new Date(time);
  // BusinessDay
  return new Date(Date.UTC(time.year, time.month - 1, time.day));
}

/**
 * Format x-axis ticks so short windows (1d / <2 calendar days) use clock time
 * instead of repeating the same day label (e.g. two "12 Agu" marks).
 */
export function formatEquityTickMark(
  time: Time,
  tickMarkType: TickMarkType,
  locale: string,
  period: PortfolioPerformancePeriod,
  spanSec: number,
): string {
  const d = timeToDate(time);
  const loc = locale || undefined;
  const shortWindow = period === '1d' || spanSec < 48 * 3600;

  if (shortWindow) {
    // Always clock time within a day — DayOfMonth marks would otherwise all say "12 Agu".
    if (tickMarkType === TickMarkType.DayOfMonth || tickMarkType === TickMarkType.Month) {
      return d.toLocaleString(loc, {
        day: 'numeric',
        month: 'short',
        hour: '2-digit',
        minute: '2-digit',
      });
    }
    return d.toLocaleTimeString(loc, { hour: '2-digit', minute: '2-digit' });
  }

  switch (tickMarkType) {
    case TickMarkType.Year:
      return String(d.getFullYear());
    case TickMarkType.Month:
      return d.toLocaleDateString(loc, { month: 'short', year: '2-digit' });
    case TickMarkType.DayOfMonth:
      return d.toLocaleDateString(loc, { day: 'numeric', month: 'short' });
    case TickMarkType.Time:
    case TickMarkType.TimeWithSeconds:
      return d.toLocaleTimeString(loc, { hour: '2-digit', minute: '2-digit' });
    default:
      return d.toLocaleDateString(loc, { day: 'numeric', month: 'short' });
  }
}

/** Crosshair / tooltip label (always precise). */
export function formatEquityCrosshairTime(time: Time, locale: string): string {
  const d = timeToDate(time);
  return d.toLocaleString(locale || undefined, {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}
