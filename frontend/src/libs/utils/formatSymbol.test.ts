import { describe, expect, it } from 'vitest';
import { formatSymbolDisplay, parseTradingPair } from './formatSymbol';

describe('parseTradingPair', () => {
  it('parses Coinbase hyphen pairs', () => {
    expect(parseTradingPair('btc-usd')).toEqual({
      raw: 'BTC-USD',
      base: 'BTC',
      quote: 'USD',
    });
    expect(parseTradingPair('ETH-USDT')).toEqual({
      raw: 'ETH-USDT',
      base: 'ETH',
      quote: 'USDT',
    });
  });

  it('parses Binance/Bybit compact pairs', () => {
    expect(parseTradingPair('BTCUSDT')).toEqual({
      raw: 'BTCUSDT',
      base: 'BTC',
      quote: 'USDT',
    });
    expect(parseTradingPair('ethusdc')).toEqual({
      raw: 'ETHUSDC',
      base: 'ETH',
      quote: 'USDC',
    });
    expect(parseTradingPair('BTCFDUSD').quote).toBe('FDUSD');
  });

  it('parses slash form and fiat/crypto quotes', () => {
    expect(parseTradingPair('BTC/USDT')).toEqual({
      raw: 'BTC/USDT',
      base: 'BTC',
      quote: 'USDT',
    });
    expect(parseTradingPair('USDTTRY')).toEqual({
      raw: 'USDTTRY',
      base: 'USDT',
      quote: 'TRY',
    });
    expect(parseTradingPair('BTCBIDR').quote).toBe('BIDR');
  });

  it('handles empty and unsplit symbols', () => {
    expect(parseTradingPair('')).toEqual({ raw: '', base: '', quote: '' });
    expect(parseTradingPair(null)).toEqual({ raw: '', base: '', quote: '' });
    expect(parseTradingPair('RLUSD')).toEqual({
      raw: 'RLUSD',
      base: 'RLUSD',
      quote: '',
    });
  });
});

describe('formatSymbolDisplay', () => {
  it('normalizes exchange formats to BASE/QUOTE', () => {
    expect(formatSymbolDisplay('BTCUSDT')).toBe('BTC/USDT');
    expect(formatSymbolDisplay('BTC-USD')).toBe('BTC/USD');
    expect(formatSymbolDisplay('eth-usdc')).toBe('ETH/USDC');
  });

  it('falls back for empty or unknown', () => {
    expect(formatSymbolDisplay('')).toBe('—');
    expect(formatSymbolDisplay(undefined)).toBe('—');
    expect(formatSymbolDisplay('WEIRDCOIN')).toBe('WEIRDCOIN');
  });
});
