/**
 * Display helpers for trading-pair symbols.
 *
 * Exchange APIs use native formats (BTCUSDT, BTC-USD). Product UI should show
 * a single professional convention: BASE/QUOTE. Raw symbols stay used for
 * API calls and route params.
 */

export type ParsedTradingPair = {
  /** Original symbol (trimmed, uppercased). */
  raw: string;
  base: string;
  /** Empty when the pair could not be split. */
  quote: string;
};

/**
 * Quote suffixes for *compact* symbols (Binance/Bybit: BTCUSDT).
 * Longest first so FDUSD wins over USD-ish tails and USDT over shorter tails.
 *
 * Note: bare `USD` is intentionally omitted here — tickers like RLUSD / TUSD
 * would false-split. Coinbase-style pairs use hyphens (BTC-USD) and are
 * handled by the hyphen branch below.
 */
const COMPACT_QUOTES = [
  'FDUSD',
  'USDT',
  'USDC',
  'BUSD',
  'TUSD',
  'USDP',
  'USDE',
  'DAI',
  'EUR',
  'GBP',
  'AUD',
  'BRL',
  'TRY',
  'BIDR',
  'IDRT',
  'UAH',
  'RUB',
  'ARS',
  'JPY',
  'CHF',
  'CAD',
  'NZD',
  'PLN',
  'CZK',
  'MXN',
  'ZAR',
  'BNB',
  'BTC',
  'ETH',
  'XRP',
  'SOL',
  'DOGE',
  'TRX',
  'TON',
] as const;

/**
 * Parse an exchange-native symbol into base + quote.
 * - Hyphenated (Coinbase): BTC-USD → base BTC, quote USD
 * - Compact (Binance/Bybit): BTCUSDT → base BTC, quote USDT
 * - Unrecognized: base = full symbol, quote = ''
 */
export function parseTradingPair(symbol: string | null | undefined): ParsedTradingPair {
  const raw = (symbol ?? '').trim().toUpperCase();
  if (!raw) {
    return { raw: '', base: '', quote: '' };
  }

  if (raw.includes('-')) {
    const [basePart, ...rest] = raw.split('-');
    const base = (basePart ?? '').trim();
    const quote = rest.join('-').trim();
    if (base && quote) {
      return { raw, base, quote };
    }
    return { raw, base: raw, quote: '' };
  }

  if (raw.includes('/')) {
    const [basePart, ...rest] = raw.split('/');
    const base = (basePart ?? '').trim();
    const quote = rest.join('/').trim();
    if (base && quote) {
      return { raw, base, quote };
    }
    return { raw, base: raw, quote: '' };
  }

  for (const q of COMPACT_QUOTES) {
    if (raw.length > q.length && raw.endsWith(q)) {
      const base = raw.slice(0, -q.length);
      // Avoid treating quote-only tickers or nonsense like "USDT" as pairs.
      if (base && base !== q) {
        return { raw, base, quote: q };
      }
    }
  }

  return { raw, base: raw, quote: '' };
}

/**
 * Professional display form: BTC/USDT, ETH/USD.
 * Falls back to the raw symbol when quote cannot be inferred.
 */
export function formatSymbolDisplay(symbol: string | null | undefined): string {
  const { base, quote, raw } = parseTradingPair(symbol);
  if (!raw) return '—';
  if (base && quote) return `${base}/${quote}`;
  return raw;
}
