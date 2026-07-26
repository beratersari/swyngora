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

export function parseDetailSearchParams(params: URLSearchParams): DetailUrlState {
  const interval = params.get('interval')?.trim() || DEFAULT_DETAIL_STATE.interval;
  const limitRaw = Number(params.get('limit'));
  const limit =
    Number.isFinite(limitRaw) && limitRaw >= LIMIT_MIN && limitRaw <= LIMIT_MAX
      ? Math.floor(limitRaw)
      : DEFAULT_DETAIL_STATE.limit;
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
