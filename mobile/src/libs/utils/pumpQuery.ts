import {
  DEFAULT_PUMP_DETAIL_DIRECTION,
  DEFAULT_PUMP_DETAIL_INTERVAL,
  DEFAULT_PUMP_DETAIL_LOOKBACK_HOURS,
  DEFAULT_PUMP_DETAIL_MAX_EVENTS,
  DEFAULT_PUMP_DETAIL_MIN_RETURN_PCT,
  DEFAULT_PUMP_SCAN_DIRECTION,
  DEFAULT_PUMP_SCAN_INTERVAL,
  DEFAULT_PUMP_SCAN_LOOKBACK_HOURS,
  DEFAULT_PUMP_SCAN_MIN_RETURN_PCT,
  DEFAULT_PUMP_SCAN_MODE,
  DEFAULT_PUMP_SCAN_SYMBOL_LIMIT,
} from '@/config/pumpConstants';
import type { MarketExchange } from '@/libs/api';
import { normalizeExchange } from './spotQuery';

export type PumpDirection = 'up' | 'down' | 'both';
export type PumpMode = 'close_return' | 'candle_body' | 'high_from_low';

export type PumpScanFilterState = {
  exchange: MarketExchange;
  quote: string;
  interval: string;
  lookbackHours: number;
  minReturnPct: number;
  direction: PumpDirection;
  mode: PumpMode;
  symbolLimit: number;
};

export type PumpDetailQueryState = {
  exchange: MarketExchange;
  symbol: string;
  interval: string;
  lookbackHours: number;
  minReturnPct: number;
  direction: PumpDirection;
  maxEvents: number;
};

export function defaultQuoteForExchange(exchange: string): string {
  return normalizeExchange(exchange) === 'coinbase' ? 'USD' : 'USDT';
}

export function defaultPumpScanFilters(
  exchange: string = 'binance',
): PumpScanFilterState {
  const ex = normalizeExchange(exchange);
  return {
    exchange: ex,
    quote: defaultQuoteForExchange(ex),
    interval: DEFAULT_PUMP_SCAN_INTERVAL,
    lookbackHours: DEFAULT_PUMP_SCAN_LOOKBACK_HOURS,
    minReturnPct: DEFAULT_PUMP_SCAN_MIN_RETURN_PCT,
    direction: DEFAULT_PUMP_SCAN_DIRECTION,
    mode: DEFAULT_PUMP_SCAN_MODE,
    symbolLimit: DEFAULT_PUMP_SCAN_SYMBOL_LIMIT,
  };
}

export function buildScanQuery(filters: PumpScanFilterState) {
  return {
    exchange: filters.exchange,
    quote: filters.quote,
    interval: filters.interval,
    lookbackHours: filters.lookbackHours,
    minReturnPct: filters.minReturnPct,
    direction: filters.direction,
    mode: filters.mode,
    symbolLimit: filters.symbolLimit,
  };
}

export function buildDetailPumpQuery(state: PumpDetailQueryState) {
  return {
    exchange: state.exchange,
    symbol: state.symbol,
    interval: state.interval || DEFAULT_PUMP_DETAIL_INTERVAL,
    lookbackHours: state.lookbackHours || DEFAULT_PUMP_DETAIL_LOOKBACK_HOURS,
    minReturnPct: state.minReturnPct || DEFAULT_PUMP_DETAIL_MIN_RETURN_PCT,
    direction: state.direction || DEFAULT_PUMP_DETAIL_DIRECTION,
    maxEvents: state.maxEvents || DEFAULT_PUMP_DETAIL_MAX_EVENTS,
  };
}
