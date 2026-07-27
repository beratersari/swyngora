/**
 * Normalize pump scan hits for the table.
 *
 * Live API shape (GET /api/v1/market/pumps/scan):
 * {
 *   symbol, exchange, interval, bestReturnPct,
 *   events: [{ openTime, returnPct, volumeRatio, ... }]
 * }
 * Return/vol/time live on the strongest event — not on the hit root.
 */

import type { PumpEventDto, PumpScanHitDto } from '@/libs/api';

export type PumpScanRow = {
  symbol: string;
  exchange: string;
  interval: string;
  returnPct: number | null;
  volumeRatio: number | null;
  openTime: string | null;
  eventCount: number;
};

/** Prefer event with largest |returnPct|; fall back to first event. */
export function pickBestEvent(events: PumpEventDto[] | undefined): PumpEventDto | undefined {
  if (!events?.length) return undefined;
  let best = events[0]!;
  let bestAbs = Math.abs(Number(best.returnPct) || 0);
  for (let i = 1; i < events.length; i++) {
    const ev = events[i]!;
    const abs = Math.abs(Number(ev.returnPct) || 0);
    if (abs > bestAbs) {
      best = ev;
      bestAbs = abs;
    }
  }
  return best;
}

export function pumpScanHitToRow(hit: PumpScanHitDto): PumpScanRow | null {
  const symbol = (hit.symbol ?? '').trim();
  if (!symbol) return null;
  const best = pickBestEvent(hit.events);
  const returnPct =
    hit.bestReturnPct != null && Number.isFinite(hit.bestReturnPct)
      ? hit.bestReturnPct
      : best?.returnPct != null && Number.isFinite(best.returnPct)
        ? best.returnPct
        : null;
  const volumeRatio =
    best?.volumeRatio != null && Number.isFinite(best.volumeRatio) ? best.volumeRatio : null;
  const openTime = best?.openTime?.trim() || null;

  return {
    symbol,
    exchange: (hit.exchange ?? '').trim() || 'binance',
    interval: (hit.interval ?? '').trim(),
    returnPct,
    volumeRatio,
    openTime,
    eventCount: hit.events?.length ?? 0,
  };
}

export function pumpScanHitsToRows(hits: PumpScanHitDto[] | undefined): PumpScanRow[] {
  if (!hits?.length) return [];
  const rows: PumpScanRow[] = [];
  for (const hit of hits) {
    const row = pumpScanHitToRow(hit);
    if (row) rows.push(row);
  }
  return rows;
}