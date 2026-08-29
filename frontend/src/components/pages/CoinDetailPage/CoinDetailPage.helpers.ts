import type {
  CandleChartMarker,
  CandleChartVertLine,
} from '@/components/molecules/CandleChartHost/CandleChartHost.types';
import { isoToUnixSeconds } from '@/components/molecules/CandleChartHost/CandleChartHost.vertLines';
import type { PumpEventDto, ScannerResult } from '@/libs/api';
import { type ApiCandle, ruleTypeShort } from '@/libs/utils';
import { semanticColors } from '@/styles/tokens';

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
/** Live pump payload for this pair only. Drops a previous coin's RTK `.data`. */
export function livePumpEventsForPair(
  resp: { exchange?: string; symbol?: string; events?: readonly PumpEventDto[] } | undefined,
  exchange: string,
  symbol: string,
): PumpEventDto[] {
  if (!resp?.events?.length) return [];
  const ex = exchange.trim().toLowerCase();
  const sym = symbol.trim().toUpperCase();
  if (resp.exchange && resp.exchange.toLowerCase() !== ex) return [];
  if (resp.symbol && resp.symbol.toUpperCase() !== sym) return [];
  return [...resp.events];
}

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
      color: up ? semanticColors.chart.up : semanticColors.chart.down,
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

/**
 * Map scanner hits for this venue/symbol → circle markers (UTC seconds).
 */
export function scannerResultsToChartMarkers(
  results: readonly ScannerResult[] | undefined,
  exchange: string,
  symbol: string,
): CandleChartMarker[] {
  if (!results?.length) return [];
  const ex = exchange.trim().toLowerCase();
  const sym = symbol.trim().toUpperCase();
  const byTime = new Map<number, string[]>();
  for (const r of results) {
    if ((r.exchange ?? '').toLowerCase() !== ex) continue;
    if ((r.symbol ?? '').toUpperCase() !== sym) continue;
    const ms = Date.parse(r.marketDataKey || r.matchedAt || '');
    if (!Number.isFinite(ms)) continue;
    const t = Math.floor(ms / 1000);
    const labels = byTime.get(t) ?? [];
    const tag = ruleTypeShort(r.ruleType);
    if (!labels.includes(tag)) labels.push(tag);
    byTime.set(t, labels);
  }
  const out: CandleChartMarker[] = [];
  for (const [time, labels] of byTime) {
    out.push({
      time,
      position: 'aboveBar',
      color: semanticColors.chart.up,
      shape: 'circle',
      text: labels.join('·'),
    });
  }
  return out.sort((a, b) => a.time - b.time);
}

/** Merge pump arrows and scanner circles; same bar keeps pump shape and appends labels. */
export function mergeChartMarkers(
  pumps: readonly CandleChartMarker[],
  signals: readonly CandleChartMarker[],
): CandleChartMarker[] {
  const byTime = new Map<number, CandleChartMarker>();
  for (const m of pumps) {
    byTime.set(m.time, { ...m });
  }
  for (const m of signals) {
    const prev = byTime.get(m.time);
    if (!prev) {
      byTime.set(m.time, { ...m });
      continue;
    }
    const extra = (m.text ?? '').trim();
    if (!extra) continue;
    const base = (prev.text ?? '').trim();
    if (base.includes(extra)) continue;
    byTime.set(m.time, { ...prev, text: base ? `${base} · ${extra}` : extra });
  }
  return [...byTime.values()].sort((a, b) => a.time - b.time);
}

/** Append off-venue bars after the last home-venue print (no invented flats). */
export function appendCandlesAfter(
  venue: readonly ApiCandle[],
  extra: readonly ApiCandle[],
): ApiCandle[] {
  if (!extra.length) return [...venue];
  let cutoff = 0;
  const last = venue[venue.length - 1];
  if (last) {
    const ms = Date.parse(last.openTime);
    if (Number.isFinite(ms)) cutoff = ms;
  }
  const tail = extra.filter((c) => {
    const t = Date.parse(c.openTime);
    return Number.isFinite(t) && t > cutoff;
  });
  if (!tail.length) return [...venue];
  return [...venue, ...tail];
}

/** ISO endTime so already-halted pairs include the last trading session. */
export function delistCandleQueryEndTime(
  delistTime?: string | null,
  nowMs = Date.now(),
): string | undefined {
  const halt = Date.parse(delistTime ?? '');
  if (!Number.isFinite(halt) || halt >= nowMs) return undefined;
  return new Date(halt + 24 * 60 * 60 * 1000).toISOString();
}

export function delistEventsToVertLines(args: {
  announcedAt?: string | null;
  delistTime?: string | null;
  announcedLabel: string;
  delistLabel: string;
  announcedColor: string;
  delistColor: string;
}): CandleChartVertLine[] {
  const out: CandleChartVertLine[] = [];
  const announced = isoToUnixSeconds(args.announcedAt);
  if (announced != null) {
    out.push({
      id: 'delist-announced',
      time: announced,
      color: args.announcedColor,
      label: args.announcedLabel,
    });
  }
  const halt = isoToUnixSeconds(args.delistTime);
  if (halt != null) {
    out.push({
      id: 'delist-halt',
      time: halt,
      color: args.delistColor,
      label: args.delistLabel,
    });
  }
  return out;
}
