import { formatChangePercent } from './formatMarket';
import { formatConvertedPrice, pairQuote, type DisplayCurrency, type FxRatesMap } from './displayCurrency';
import { formatSymbolDisplay } from './formatSymbol';

export type TickerTapeItem = {
  exchange: string;
  symbol: string;
  lastPrice: string;
  changePercent: string;
  changeValue: number | null;
  href: string;
};

function parseChange(value: string | number | null | undefined): number | null {
  if (value === null || value === undefined || value === '') return null;
  const n = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(n) ? n : null;
}

export function toTickerTapeItem(
  row: {
    exchange?: string;
    symbol?: string;
    lastPrice?: string | number | null;
    priceChangePercent?: string | number | null;
  },
  display?: { currency: DisplayCurrency; rates?: FxRatesMap },
): TickerTapeItem | null {
  const exchange = (row.exchange ?? '').trim().toLowerCase();
  const symbol = (row.symbol ?? '').trim().toUpperCase();
  if (!exchange || !symbol) return null;
  const pref = display?.currency ?? 'native';
  return {
    exchange,
    symbol,
    lastPrice: formatConvertedPrice(row.lastPrice, pairQuote(symbol, exchange), pref, display?.rates),
    changePercent: formatChangePercent(row.priceChangePercent),
    changeValue: parseChange(row.priceChangePercent),
    href: `/markets/${exchange}/${encodeURIComponent(symbol)}`,
  };
}

export function tapeCellLabel(item: TickerTapeItem): string {
  return `${formatSymbolDisplay(item.symbol)} ${item.lastPrice} ${item.changePercent}`;
}
