import type { ChartLinePoint } from './indicators';

export type ComparePair = {
  exchange: string;
  symbol: string;
};

export const MAX_COMPARE_PAIRS = 3;

/** Normalize closes to % change from the first finite close. */
export function closesToPercentSeries(
  points: Array<{ time: number; close: number }>,
): ChartLinePoint[] {
  const finite = points.filter(
    (p) => Number.isFinite(p.time) && Number.isFinite(p.close) && p.close !== 0,
  );
  if (!finite.length) return [];
  const base = finite[0]!.close;
  return finite.map((p) => ({
    time: p.time,
    value: ((p.close - base) / base) * 100,
  }));
}

/** Parse `binance:BTCUSDT,coinbase:BTC-USD` style list (max MAX_COMPARE_PAIRS). */
export function parseComparePairsParam(raw: string | null | undefined): ComparePair[] {
  if (!raw?.trim()) return [];
  const out: ComparePair[] = [];
  const seen = new Set<string>();
  for (const part of raw.split(',')) {
    const token = part.trim();
    if (!token) continue;
    const colon = token.indexOf(':');
    if (colon <= 0) continue;
    const exchange = token.slice(0, colon).toLowerCase().trim();
    const symbol = token.slice(colon + 1).trim().toUpperCase();
    if (!exchange || !symbol) continue;
    const key = `${exchange}:${symbol}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push({ exchange, symbol });
    if (out.length >= MAX_COMPARE_PAIRS) break;
  }
  return out;
}

export function serializeComparePairs(pairs: readonly ComparePair[]): string {
  return pairs
    .slice(0, MAX_COMPARE_PAIRS)
    .map((p) => `${p.exchange.toLowerCase()}:${p.symbol.toUpperCase()}`)
    .join(',');
}

export function comparePairKey(p: ComparePair): string {
  return `${p.exchange.toLowerCase()}:${p.symbol.toUpperCase()}`;
}
