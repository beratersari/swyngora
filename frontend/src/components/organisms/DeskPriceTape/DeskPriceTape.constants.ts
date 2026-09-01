import type { DeskTapeSource } from './DeskPriceTape.types';

export { DESK_TAPE_SOURCE_STORAGE_KEY, DESK_TAPE_VENUE_LIMIT } from '@/libs/utils';

export const DESK_TAPE_SOURCE_LABEL_KEY = {
  binance: 'exchanges.binance',
  coinbase: 'exchanges.coinbase',
  bist: 'exchanges.bist',
  watchlist: 'nav.watchlist',
} as const satisfies Record<DeskTapeSource, string>;
