export const TAPE_CONDITIONS = ['cvd', 'liquidation', 'book', 'funding', 'oi', 'gap', 'size'] as const;

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
  funding: { side: 'long', threshold: 0.01 },
  oi: { side: 'up', threshold: 5 },
  gap: { side: 'top', threshold: 3 },
  size: { side: 'whale', threshold: 0.4 },
} as const;

export const TAPE_EXCHANGES = {
  cvd: ['all', 'binance', 'bybit', 'okx'],
  liquidation: ['all', 'binance', 'bybit', 'okx'],
  book: ['all', 'binance', 'coinbase', 'bybit'],
  funding: ['all', 'binance', 'bybit', 'okx'],
  oi: ['all', 'binance', 'bybit', 'okx'],
  gap: ['all', 'binance', 'okx'],
  size: ['all', 'binance', 'bybit', 'okx'],
} as const;

export const TAPE_SIDES = {
  cvd: ['buy', 'sell'],
  liquidation: ['long', 'short'],
  book: ['bid', 'ask'],
  funding: ['long', 'short'],
  oi: ['up', 'down'],
  gap: ['top', 'crowd'],
  size: ['whale', 'retail'],
} as const;

export const TAPE_THRESHOLD_BOUNDS = {
  cvd: { min: 0.55, max: 0.9, step: 0.01 },
  liquidation: { min: 2, max: 12, step: 0.5 },
  book: { min: 0.55, max: 0.9, step: 0.01 },
  funding: { min: 0.005, max: 0.2, step: 0.005 },
  oi: { min: 1, max: 50, step: 1 },
  gap: { min: 1, max: 20, step: 1 },
  size: { min: 0.15, max: 0.8, step: 0.01 },
} as const;
