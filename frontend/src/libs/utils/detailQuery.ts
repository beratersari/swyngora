import {
  DEFAULT_DETAIL_CANDLE_LIMIT,
  DEFAULT_DETAIL_INTERVAL,
  SUPPORTED_EXCHANGES,
  type SupportedExchange,
} from '@/config/constants';

export const DETAIL_TABS = [
  'overview',
  'orderbook',
  'tape',
  'holders',
  'indicators',
  'trade',
] as const;
export type DetailTab = (typeof DETAIL_TABS)[number];
export const DEFAULT_DETAIL_TAB: DetailTab = 'overview';

export type DetailUrlState = {
  interval: string;
  /**
   * @deprecated Bars are no longer URL-controlled; progressive load owns limit.
   * Still parsed if present for old links, then ignored by the page.
   */
  limit: number;
  tab: DetailTab;
};

export const DEFAULT_DETAIL_STATE: DetailUrlState = {
  interval: DEFAULT_DETAIL_INTERVAL,
  limit: DEFAULT_DETAIL_CANDLE_LIMIT,
  tab: DEFAULT_DETAIL_TAB,
};

export function parseDetailTab(raw: string | null | undefined): DetailTab {
  const v = (raw ?? '').trim().toLowerCase();
  return (DETAIL_TABS as readonly string[]).includes(v) ? (v as DetailTab) : DEFAULT_DETAIL_TAB;
}

const LIMIT_MIN = 20;
/** Legacy URL limit clamp only (chart history is no longer URL-driven). */
const LIMIT_MAX = 1000;

/**
 * Interval string → duration in seconds (e.g. `15m` → 900, `1h` → 3600).
 * Returns 0 when the token is not a simple m/h/d unit.
 */
export function intervalToSeconds(interval: string): number {
  // Case-sensitive units: `1m` minute vs Binance `1M` month (unsupported here → 0).
  const m = /^(\d+)([mhdw])$/.exec((interval ?? '').trim());
  if (!m) return 0;
  const n = Number(m[1]);
  if (!Number.isFinite(n) || n <= 0) return 0;
  switch (m[2]) {
    case 'm':
      return n * 60;
    case 'h':
      return n * 3600;
    case 'd':
      return n * 86400;
    case 'w':
      return n * 604800;
    default:
      return 0;
  }
}

/**
 * Candle count for pump/indicator APIs: track loaded chart depth up to the
 * backend max (1000). Never below `minLive` so the first paint still analyzes
 * a full live window.
 */
export function analyticsBarLimit(
  loadedBars: number,
  minLive: number,
  apiMax = 1000,
): number {
  const floor = Math.max(1, Math.min(apiMax, minLive));
  if (!Number.isFinite(loadedBars) || loadedBars <= 0) return floor;
  return Math.min(apiMax, Math.max(floor, Math.floor(loadedBars)));
}

/**
 * Parse a path/query exchange token.
 * Returns null when missing or not a supported venue (caller should show an error).
 */
export function parseExchangeParam(raw: string | undefined): SupportedExchange | null {
  const v = (raw ?? '').trim().toLowerCase();
  if (!v) return null;
  if ((SUPPORTED_EXCHANGES as readonly string[]).includes(v)) {
    return v as SupportedExchange;
  }
  return null;
}

/** Like parseExchangeParam but falls back to binance for list defaults only. */
export function parseExchangeParamOrDefault(
  raw: string | undefined,
  fallback: SupportedExchange = 'binance',
): SupportedExchange {
  return parseExchangeParam(raw) ?? fallback;
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
const SUPPLY_QUOTE_SUFFIXES = [
  'FDUSD',
  'USDT',
  'USDC',
  'BUSD',
  'TUSD',
  'DAI',
  'EUR',
  'TRY',
  'BRL',
  'GBP',
  'AUD',
  'CAD',
  'ARS',
  'JPY',
  'BTC',
  'ETH',
  'BNB',
] as const;

const CRYPTO_QUOTE_SUFFIXES = new Set(['BTC', 'ETH', 'BNB']);

/** Left sides of BTC/ETH/BNB pairs. Wrapped tails (WBTC, STETH) are omitted. */
const CRYPTO_PAIR_BASES = new Set([
  'BTC',
  'ETH',
  'BNB',
  'SOL',
  'XRP',
  'ADA',
  'DOGE',
  'AVAX',
  'DOT',
  'LINK',
  'ATOM',
  'LTC',
  'BCH',
  'UNI',
  'APT',
  'SUI',
  'NEAR',
  'FIL',
  'INJ',
  'TIA',
  'SEI',
  'OP',
  'ARB',
  'AAVE',
  'MKR',
  'LDO',
  'TRX',
  'TON',
  'HBAR',
  'ICP',
  'XLM',
  'ETC',
  'APE',
  'PEPE',
  'SHIB',
  'WIF',
  'BONK',
  'JUP',
  'FET',
  'GRT',
  'SAND',
  'MANA',
  'CRV',
  'SNX',
  'IMX',
]);

export function toSupplyAsset(symbol: string): string {
  const s = symbol.trim().toUpperCase();
  if (!s) return '';
  // Coinbase / unified: BASE-QUOTE
  if (s.includes('-')) {
    const base = s.split('-')[0]?.trim() ?? '';
    return base || s;
  }
  for (const q of SUPPLY_QUOTE_SUFFIXES) {
    if (s.length > q.length && s.endsWith(q)) {
      const base = s.slice(0, -q.length);
      if (!base || base === q) continue;
      if (CRYPTO_QUOTE_SUFFIXES.has(q) && !CRYPTO_PAIR_BASES.has(base)) continue;
      return base;
    }
  }
  return s;
}

/** Map a spot pair onto the USD-M / linear perp ticker used by OI, funding, CVD, liqs. */
export function toPerpSymbol(symbol: string): string {
  const s = symbol.trim().toUpperCase();
  if (!s) return '';
  if (s.includes('-')) {
    const [base, quote] = s.split('-');
    if (!base) return s;
    const q = !quote || quote === 'USD' ? 'USDT' : quote;
    return `${base}${q}`;
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
  return { interval, limit, tab: parseDetailTab(params.get('tab')) };
}

/** Serialize detail URL state. Limit is not written (scroll-loads history in-app). */
export function detailStateToSearchParams(
  state: Pick<DetailUrlState, 'interval'> &
    Partial<Pick<DetailUrlState, 'limit' | 'tab'>>,
): URLSearchParams {
  const p = new URLSearchParams();
  if (state.interval && state.interval !== DEFAULT_DETAIL_STATE.interval) {
    p.set('interval', state.interval);
  }
  if (state.tab && state.tab !== DEFAULT_DETAIL_TAB) {
    p.set('tab', state.tab);
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
