import { MAX_WATCHLIST_ITEMS } from '@/config/watchlistConstants';
import { normalizePair, watchKey, type WatchlistPair } from './watchlistKey';

/**
 * Union by exchange|symbol. When both present, **local wins** so optimistic
 * offline adds are not wiped by an empty or lagging server list.
 */
export function mergeWatchlists(
  local: WatchlistPair[],
  server: WatchlistPair[],
): WatchlistPair[] {
  const map = new Map<string, WatchlistPair>();
  for (const w of server ?? []) {
    const p = normalizePair(w.exchange, w.symbol);
    if (!p.symbol) continue;
    map.set(watchKey(p.exchange, p.symbol), { ...p, note: w.note });
  }
  for (const w of local ?? []) {
    const p = normalizePair(w.exchange, w.symbol);
    if (!p.symbol) continue;
    map.set(watchKey(p.exchange, p.symbol), { ...p, note: w.note });
  }
  return Array.from(map.values());
}

export function isAtMaxItems(
  items: WatchlistPair[],
  max: number = MAX_WATCHLIST_ITEMS,
): boolean {
  return items.length >= max;
}

export function readLocalWatchlist(raw: string | null): WatchlistPair[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed
      .map((row) => {
        if (!row || typeof row !== 'object') return null;
        const r = row as { exchange?: string; symbol?: string; note?: string };
        const p = normalizePair(r.exchange ?? 'binance', r.symbol ?? '');
        if (!p.symbol) return null;
        return r.note ? { ...p, note: String(r.note) } : p;
      })
      .filter((x): x is WatchlistPair => x !== null);
  } catch {
    return [];
  }
}

export function serializeLocalWatchlist(items: WatchlistPair[]): string {
  return JSON.stringify(
    items.map((i) => ({
      exchange: i.exchange,
      symbol: i.symbol,
      ...(i.note ? { note: i.note } : {}),
    })),
  );
}
