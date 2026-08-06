import type { LineData, Time } from 'lightweight-charts';
import type { IndicatorChartHostProps } from './IndicatorChartHost.types';

export function toRsiLineData(data: IndicatorChartHostProps['data']): LineData<Time>[] {
  if (!data.length) return [];
  const sorted = [...data]
    .filter((p) => Number.isFinite(p.time) && Number.isFinite(p.value))
    .sort((a, b) => a.time - b.time);
  const out: LineData<Time>[] = [];
  let lastTime: number | null = null;
  for (const p of sorted) {
    if (lastTime !== null && p.time === lastTime) {
      out[out.length - 1] = { time: p.time as Time, value: p.value };
      continue;
    }
    out.push({ time: p.time as Time, value: p.value });
    lastTime = p.time;
  }
  return out;
}
