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
      if (!base || base === q) continue;
      if (CRYPTO_QUOTE_SUFFIXES.has(q) && !CRYPTO_PAIR_BASES.has(base)) continue;
      return base;
    }
  }
  return s;
}
