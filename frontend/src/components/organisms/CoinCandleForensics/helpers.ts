import { COIN_FORENSICS_DURING, COIN_FORENSICS_EXCHANGE, COIN_FORENSICS_WINDOW } from './constants';

/** Full forensics desk for the same symbol, candle open, window, and during. */
export function coinForensicsHref(symbol: string, atIso: string): string {
  const params = new URLSearchParams({
    view: 'event',
    symbol,
    at: atIso,
    exchange: COIN_FORENSICS_EXCHANGE,
    window: COIN_FORENSICS_WINDOW,
    during: COIN_FORENSICS_DURING,
  });
  return `/forensics?${params.toString()}`;
}

/** Unix seconds for a similar-move time. Invalid text stays off the chart. */
export function clickSecFromForensicsAt(at: string): number | null {
  const ms = Date.parse(at);
  if (!Number.isFinite(ms)) return null;
  return Math.floor(ms / 1000);
}
