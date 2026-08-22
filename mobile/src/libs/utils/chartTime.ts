/** Axis + crosshair show clock time. LWC defaults to date-only. */
export const CHART_TIME_SCALE = {
  timeVisible: true,
  secondsVisible: true,
} as const;

function pad2(n: number): string {
  return String(n).padStart(2, '0');
}

export function formatChartDateTime(time: number | string | { year: number; month: number; day: number }): string {
  let d: Date | null = null;
  if (typeof time === 'number' && Number.isFinite(time)) {
    d = new Date(time > 1e12 ? time : time * 1000);
  } else if (typeof time === 'string') {
    d = new Date(time);
  } else if (time && typeof time === 'object') {
    d = new Date(time.year, time.month - 1, time.day);
  }
  if (!d || !Number.isFinite(d.getTime())) return '';
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}`;
}

export const CHART_LOCALIZATION = {
  timeFormatter: formatChartDateTime,
} as const;
