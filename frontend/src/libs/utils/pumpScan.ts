/**
 * Normalize pump scan hits for the table.
 *
 * Live API shape (GET /api/v1/market/pumps/scan):
 * {
 *   symbol, exchange, interval, bestReturnPct, bestVolumeRatio, bestOpenTime,
 *   events: [...]
 * }
 * Best-event fields are computed by the API.
 */

import type { PumpScanHitDto } from '@/libs/api';

export type PumpScanRow = {
  symbol: string;
  exchange: string;
  interval: string;
  returnPct: number | null;
  volumeRatio: number | null;
  openTime: string | null;
  eventCount: number;
};

export function pumpScanHitToRow(hit: PumpScanHitDto): PumpScanRow | null {
  const symbol = (hit.symbol ?? '').trim();
  if (!symbol) return null;
  const returnPct =
    hit.bestReturnPct != null && Number.isFinite(hit.bestReturnPct) ? hit.bestReturnPct : null;
  const volumeRatio =
    hit.bestVolumeRatio != null && Number.isFinite(hit.bestVolumeRatio)
      ? hit.bestVolumeRatio
      : null;
  const openTime = hit.bestOpenTime?.trim() || null;

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