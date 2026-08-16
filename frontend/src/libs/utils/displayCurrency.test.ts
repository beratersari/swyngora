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

  it('maps venue quotes', () => {
    expect(venueQuote('bist')).toBe('TRY');
    expect(venueQuote('nasdaq')).toBe('USD');
    expect(venueQuote('binance')).toBe('USDT');
    expect(marketCapQuote('bist')).toBe('TRY');
    expect(marketCapQuote('nasdaq')).toBe('USD');
  });

  it('converts TRY last price to USD', () => {
    expect(convertAmount(400, 'TRY', 'USD', rates)).toBe(10);
    expect(convertAmount(10, 'USD', 'TRY', rates)).toBe(400);
    expect(convertAmount(1, 'USDT', 'TRY', rates)).toBe(40);
  });

  it('aliases USDT to USD', () => {
    expect(aliasFxCode('usdt')).toBe('USD');
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
    expect(pairQuote('AAPL', 'nasdaq')).toBe('USD');
    expect(formatConvertedPrice(0.035, pairQuote('ETHBTC', 'binance'), 'native', rates)).toMatch(/BTC/);
    expect(formatConvertedPrice(0.035, pairQuote('ETHBTC', 'binance'), 'USD', rates)).toBe('—');
  });
});
