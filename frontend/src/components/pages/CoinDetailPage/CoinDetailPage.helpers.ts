import type { CandleChartMarker } from '@/components/molecules/CandleChartHost/CandleChartHost.types';
import type { PumpEventDto } from '@/libs/api';
import { palette, semanticColors } from '@/styles/tokens';

/**
 * Merge pump events from live + history pages by openTime.
 * Keeps the stronger |returnPct| when the same bar appears twice.
 */
export function mergePumpEvents(
  ...lists: Array<readonly PumpEventDto[] | undefined>
): PumpEventDto[] {
  const byTime = new Map<string, PumpEventDto>();
  for (const list of lists) {
    if (!list?.length) continue;
    for (const ev of list) {
      const key = ev.openTime?.trim();
      if (!key) continue;
      const prev = byTime.get(key);
      if (!prev) {
        byTime.set(key, ev);
        continue;
      }
      const prevAbs = Math.abs(Number(prev.returnPct) || 0);
      const nextAbs = Math.abs(Number(ev.returnPct) || 0);
      if (nextAbs >= prevAbs) byTime.set(key, ev);
    }
  }
  return [...byTime.values()].sort(
    (a, b) => Date.parse(a.openTime ?? '') - Date.parse(b.openTime ?? ''),
  );
}

/**
 * Map pump API events → chart markers (UTC seconds).
 * Keeps dumps (negative returnPct) and filters by |return| threshold.
 */
export function pumpEventsToChartMarkers(
  events: readonly PumpEventDto[] | undefined,
  minAbsReturnPct: number,
): CandleChartMarker[] {
  if (!events?.length) return [];
  const thr =
    Number.isFinite(minAbsReturnPct) && minAbsReturnPct > 0 ? minAbsReturnPct : 0;
  const out: CandleChartMarker[] = [];
  for (const ev of events) {
    if (!ev.openTime) continue;
    const ms = Date.parse(ev.openTime);
    if (!Number.isFinite(ms)) continue;
    const ret = Number(ev.returnPct);
    if (Number.isFinite(ret) && thr > 0 && Math.abs(ret) < thr) continue;
    const up = !Number.isFinite(ret) || ret >= 0;
    out.push({
      time: Math.floor(ms / 1000),
      position: up ? 'belowBar' : 'aboveBar',
      color: up ? palette.mountainMeadow : semanticColors.chart.down,
      shape: up ? 'arrowUp' : 'arrowDown',
      text: up
        ? `↑${Number.isFinite(ret) ? ret.toFixed(1) : ''}`
        : `↓${Number.isFinite(ret) ? Math.abs(ret).toFixed(1) : ''}`,
    });
  }
  // Prefer stronger moves first so snapMarkers one-per-bar keeps the larger event.
  out.sort((a, b) => {
    const ra = Math.abs(parseFloat((a.text ?? '').replace(/[↑↓]/g, '')) || 0);
    const rb = Math.abs(parseFloat((b.text ?? '').replace(/[↑↓]/g, '')) || 0);
    return rb - ra;
  });
  return out;
}
