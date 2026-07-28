import type { MarketExchange } from '@/libs/api';

/** Default venue for Home list widgets. */
export const HOME_DEFAULT_EXCHANGE: MarketExchange = 'binance';

/** Default quote filter for Home spot widgets. */
export const HOME_DEFAULT_QUOTE = 'USDT';

export const HOME_MOVERS_LIMIT = 5;
export const HOME_VOLUME_LIMIT = 5;
export const HOME_FAVORITES_LIMIT = 5;
export const HOME_PUMP_TEASER_LIMIT = 3;

/** Poll while Home is focused and app active. */
export const HOME_WIDGET_POLL_MS = 20_000;
