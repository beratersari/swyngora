export const TAPE_CONDITIONS = ['cvd', 'liquidation', 'book'] as const;

export const TAPE_RANGES = [
  { days: 1, key: '1d' },
  { days: 3, key: '3d' },
  { days: 7, key: '7d' },
] as const;

export const TAPE_DEFAULT_SYMBOL = 'BTCUSDT';

export const TAPE_DEFAULTS = {
  cvd: { side: 'buy', threshold: 0.65 },
  liquidation: { side: 'long', threshold: 4 },
  book: { side: 'bid', threshold: 0.65 },
} as const;

export const TAPE_EXCHANGES = {
  cvd: ['all', 'binance', 'bybit', 'okx'],
  liquidation: ['all', 'binance', 'bybit', 'okx'],
  book: ['all', 'binance', 'coinbase', 'bybit'],
} as const;

export const TAPE_SIDES = {
  cvd: ['buy', 'sell'],
  liquidation: ['long', 'short'],
  book: ['bid', 'ask'],
} as const;
