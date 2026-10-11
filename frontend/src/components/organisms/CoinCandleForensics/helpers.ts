import { COIN_FORENSICS_DURING, COIN_FORENSICS_EXCHANGE, COIN_FORENSICS_WINDOW } from './constants';

const DURING_MS: Record<string, number> = {
  '5m': 5 * 60 * 1000,
  '15m': 15 * 60 * 1000,
  '30m': 30 * 60 * 1000,
  '1h': 60 * 60 * 1000,
};

/**
 * Observer clock for a past event: the end of the during window.
 * A window that has not finished yet returns undefined so the server uses now.
 * Later prints stay out of an old candle. The during minutes stay in.
 */
export function forensicsKnownUntil(
  atIso: string,
  during: string,
  nowMs: number = Date.now(),
): string | undefined {
  const at = Date.parse(atIso);
  const span = DURING_MS[during];
  if (!Number.isFinite(at) || span == null) return undefined;
  const end = at + span;
  if (end >= nowMs) return undefined;
  return new Date(end).toISOString();
}

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
