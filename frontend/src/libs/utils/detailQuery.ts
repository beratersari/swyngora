import {
  DEFAULT_DETAIL_CANDLE_LIMIT,
  DEFAULT_DETAIL_INTERVAL,
  SUPPORTED_EXCHANGES,
  type SupportedExchange,
} from '@/config/constants';

export type DetailUrlState = {
  interval: string;
  limit: number;
};

export const DEFAULT_DETAIL_STATE: DetailUrlState = {
  interval: DEFAULT_DETAIL_INTERVAL,
  limit: DEFAULT_DETAIL_CANDLE_LIMIT,
};

const LIMIT_MIN = 20;
const LIMIT_MAX = 500;

export function parseExchangeParam(raw: string | undefined): SupportedExchange {
  const v = (raw ?? '').toLowerCase();
  if ((SUPPORTED_EXCHANGES as readonly string[]).includes(v)) {
    return v as SupportedExchange;
  }
  return 'binance';
}

/** Decode path symbol (Coinbase uses BTC-USD). */
export function parseSymbolParam(raw: string | undefined): string {
  if (!raw) return '';
  try {
    return decodeURIComponent(raw).trim().toUpperCase();
  } catch {
    return raw.trim().toUpperCase();
  }
}

/**
 * Map a trading pair symbol to the supply-cache asset key (base ticker).
 * Backend supply is asset-level (e.g. BTC); it strips unhyphenated stables
 * (BTCUSDT → BTC) but not Coinbase-style `BASE-QUOTE` (BTC-USD stays as-is → 404).
 * Longest stable first so USDT/FDUSD win over shorter tails.
 */
const SUPPLY_STABLE_QUOTES = ['FDUSD', 'USDT', 'USDC', 'BUSD', 'TUSD', 'DAI'] as const;

export function toSupplyAsset(symbol: string): string {
  const s = symbol.trim().toUpperCase();
  if (!s) return '';
  // Coinbase / unified: BASE-QUOTE
  if (s.includes('-')) {
    const base = s.split('-')[0]?.trim() ?? '';
    return base || s;
  }
  for (const q of SUPPLY_STABLE_QUOTES) {
    if (s.length > q.length && s.endsWith(q)) {
      const base = s.slice(0, -q.length);
      if (base && base !== q) return base;
    }
  }
  return s;
}

/** Markets list URL that preserves the venue the user was browsing. */
export function marketsBackPath(exchange: string): string {
  const p = new URLSearchParams();
  const ex = (exchange || '').toLowerCase();
  if (ex && ex !== 'binance') {
    p.set('exchange', ex);
  }
  const qs = p.toString();
  return qs ? `/markets?${qs}` : '/markets';
}

export function parseDetailSearchParams(params: URLSearchParams): DetailUrlState {
  const interval = params.get('interval')?.trim() || DEFAULT_DETAIL_STATE.interval;
  const limitRaw = Number(params.get('limit'));
  // Floor first, then clamp — same pattern as markets parseIntParam
  // so limit=500.1 → 500 (not default), limit=19.9 → default (below min after floor).
  let limit = DEFAULT_DETAIL_STATE.limit;
  if (Number.isFinite(limitRaw)) {
    const floored = Math.floor(limitRaw);
    if (floored >= LIMIT_MIN && floored <= LIMIT_MAX) {
      limit = floored;
    }
  }
  return { interval, limit };
}

export function detailStateToSearchParams(state: DetailUrlState): URLSearchParams {
  const p = new URLSearchParams();
  if (state.interval && state.interval !== DEFAULT_DETAIL_STATE.interval) {
    p.set('interval', state.interval);
  }
  if (state.limit !== DEFAULT_DETAIL_STATE.limit) {
    p.set('limit', String(state.limit));
  }
  return p;
}

/** Prefer requested interval; fall back to first supported or default. */
export function resolveInterval(
  requested: string,
  supported: string[] | undefined,
): string {
  if (supported?.includes(requested)) return requested;
  if (supported?.includes(DEFAULT_DETAIL_INTERVAL)) return DEFAULT_DETAIL_INTERVAL;
  if (supported?.length) return supported[0]!;
  return requested || DEFAULT_DETAIL_INTERVAL;
}
