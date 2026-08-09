export const REALTIME_PATH = '/api/v1/ws';

export const REALTIME_RECONNECT_MIN_MS = 500;
export const REALTIME_RECONNECT_MAX_MS = 8_000;
export const REALTIME_PING_MS = 25_000;

export const REALTIME_OP = {
  subscribePrices: 'subscribe_prices',
  unsubscribePrices: 'unsubscribe_prices',
  subscribePortfolio: 'subscribe_portfolio',
  unsubscribePortfolio: 'unsubscribe_portfolio',
  ping: 'ping',
} as const;

export const REALTIME_TYPE = {
  hello: 'hello',
  ack: 'ack',
  pong: 'pong',
  error: 'error',
  price: 'price',
  portfolio: 'portfolio',
} as const;
