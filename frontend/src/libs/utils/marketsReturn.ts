/** Session key for last Markets list URL (filters/sort/page) for detail back nav. */
export const MARKETS_RETURN_STORAGE_KEY = 'swyngora.markets.return';

/**
 * Remember the current markets list query string so coin detail can return
 * to the same exchange/quote/tag/search/sort/offset.
 * Pass the result of `location.search` (including leading `?`) or raw qs.
 */
export function rememberMarketsReturnPath(search: string): void {
  if (typeof sessionStorage === 'undefined') return;
  try {
    const raw = (search ?? '').trim();
    const normalized =
      !raw || raw === '?'
        ? ''
        : raw.startsWith('?')
          ? raw
          : `?${raw}`;
    sessionStorage.setItem(MARKETS_RETURN_STORAGE_KEY, normalized);
  } catch {
    // private mode / quota — ignore
  }
}

/**
 * Markets list URL that prefers the remembered filter state, else venue only.
 */
export function marketsBackPath(exchange: string): string {
  if (typeof sessionStorage !== 'undefined') {
    try {
      const saved = sessionStorage.getItem(MARKETS_RETURN_STORAGE_KEY);
      if (saved !== null) {
        return saved ? `/markets${saved}` : '/markets';
      }
    } catch {
      // fall through
    }
  }
  const p = new URLSearchParams();
  const ex = (exchange || '').toLowerCase();
  if (ex && ex !== 'binance') {
    p.set('exchange', ex);
  }
  const qs = p.toString();
  return qs ? `/markets?${qs}` : '/markets';
}
