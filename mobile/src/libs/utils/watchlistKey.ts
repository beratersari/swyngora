/** Normalized membership key: `exchange|SYMBOL` */
export function watchKey(exchange: string, symbol: string): string {
  return `${String(exchange || 'binance').toLowerCase()}|${String(symbol || '').toUpperCase()}`;
}

export type WatchlistPair = {
  exchange: string;
  symbol: string;
  note?: string;
};

export function normalizePair(exchange: string, symbol: string): WatchlistPair {
  return {
    exchange: String(exchange || 'binance').toLowerCase(),
    symbol: String(symbol || '').toUpperCase(),
  };
}
