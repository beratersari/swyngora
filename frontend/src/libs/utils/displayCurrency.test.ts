import { describe, expect, it } from 'vitest';
import {
  aliasFxCode,
  convertAmount,
  formatConvertedPrice,
  marketCapQuote,
  pairQuote,
  scalePriceSeries,
  venueQuote,
} from './displayCurrency';

describe('displayCurrency', () => {
  const rates = { TRY: 40, EUR: 0.9, USD: 1, USDT: 1 };
  const venues = { bist: 'TRY', nasdaq: 'USD', binance: 'USDT', coinbase: 'USD' };
  const mcaps = { bist: 'TRY', nasdaq: 'USD', binance: 'USD' };
  const aliases = { USDT: 'USD', USDC: 'USD', BUSD: 'USD' };

  it('reads venue quotes only from the API map', () => {
    expect(venueQuote('bist')).toBe('');
    expect(venueQuote('bist', venues)).toBe('TRY');
    expect(venueQuote('binance', venues)).toBe('USDT');
    expect(marketCapQuote('bist')).toBe('');
    expect(marketCapQuote('bist', mcaps)).toBe('TRY');
  });

  it('converts TRY last price to USD', () => {
    expect(convertAmount(400, 'TRY', 'USD', rates, aliases)).toBe(10);
    expect(convertAmount(10, 'USD', 'TRY', rates, aliases)).toBe(400);
    expect(convertAmount(1, 'USDT', 'TRY', rates, aliases)).toBe(40);
  });

  it('aliases only from the API map', () => {
    expect(aliasFxCode('usdt')).toBe('USDT');
    expect(aliasFxCode('usdt', aliases)).toBe('USD');
  });

  it('formats native and converted prices with a code', () => {
    expect(formatConvertedPrice(190.5, 'USD', 'native', rates)).toMatch(/USD/);
    expect(formatConvertedPrice(400, 'TRY', 'USD', rates)).toMatch(/10/);
    expect(formatConvertedPrice(400, 'TRY', 'USD', rates)).toMatch(/USD/);
  });

  it('scales EMA/MA overlay points with the same FX as candles', () => {
    const line = [
      { time: 1, value: 400 },
      { time: 2, value: 800 },
    ];
    expect(scalePriceSeries(line, 'TRY', 'native', rates)).toEqual(line);
    expect(scalePriceSeries(line, 'TRY', 'USD', rates)).toEqual([
      { time: 1, value: 10 },
      { time: 2, value: 20 },
    ]);
  });

  it('pairQuote uses the symbol quote, not the venue default', () => {
    expect(pairQuote('ETHBTC', 'binance')).toBe('BTC');
    expect(pairQuote('BTCEUR', 'binance')).toBe('EUR');
    expect(pairQuote('BTCUSDT', 'binance')).toBe('USDT');
    expect(pairQuote('AAPL', 'nasdaq', venues)).toBe('USD');
    expect(formatConvertedPrice(0.035, pairQuote('ETHBTC', 'binance'), 'native', rates)).toMatch(/BTC/);
    expect(formatConvertedPrice(0.035, pairQuote('ETHBTC', 'binance'), 'USD', rates)).toBe('—');
  });
});
