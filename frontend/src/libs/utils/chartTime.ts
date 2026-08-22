import type { Time } from 'lightweight-charts';

/** Axis + crosshair show clock time. LWC defaults to date-only (`timeVisible: false`). */
export const CHART_TIME_SCALE = {
  timeVisible: true,
  secondsVisible: true,
} as const;

function pad2(n: number): string {
  return String(n).padStart(2, '0');
}

/** Local wall-clock `YYYY-MM-DD HH:mm:ss` for a Lightweight Charts time value. */
export function formatChartDateTime(time: Time): string {
  const d = chartTimeToDate(time);
  if (!d) return '';
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}`;
}

export function chartTimeToDate(time: Time): Date | null {
  if (typeof time === 'number' && Number.isFinite(time)) {
    const ms = time > 1e12 ? time : time * 1000;
    const d = new Date(ms);
    return Number.isFinite(d.getTime()) ? d : null;
  }
  if (typeof time === 'string') {
    const d = new Date(time);
    return Number.isFinite(d.getTime()) ? d : null;
  }
  if (time && typeof time === 'object' && 'year' in time) {
    const d = new Date(time.year, time.month - 1, time.day);
    return Number.isFinite(d.getTime()) ? d : null;
  }
  return null;
}

export const CHART_LOCALIZATION = {
  timeFormatter: formatChartDateTime,
} as const;
