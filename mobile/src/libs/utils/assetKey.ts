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

/** Map a trading pair to the supply/holders catalog key (BTC-USD / ETHTRY → BTC). */
export function toSupplyAsset(symbol: string): string {
  const s = symbol.trim().toUpperCase();
  if (!s) return '';
  if (s.includes('-')) {
    const base = s.split('-')[0]?.trim() ?? '';
    return base || s;
  }
  for (const q of SUPPLY_QUOTE_SUFFIXES) {
    if (s.length > q.length && s.endsWith(q)) {
      const base = s.slice(0, -q.length);
      if (base && base !== q) return base;
    }
  }
  return s;
}
