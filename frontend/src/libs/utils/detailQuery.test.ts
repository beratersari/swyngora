import { describe, expect, it } from 'vitest';
import {
  detailStateToSearchParams,
  parseDetailSearchParams,
  parseExchangeParam,
  parseSymbolParam,
  resolveInterval,
} from './detailQuery';

describe('parseExchangeParam', () => {
  it('accepts known venues and defaults unknown', () => {
    expect(parseExchangeParam('coinbase')).toBe('coinbase');
    expect(parseExchangeParam('NOPE')).toBe('binance');
  });
});

describe('parseSymbolParam', () => {
  it('decodes and uppercases', () => {
    expect(parseSymbolParam('btc-usd')).toBe('BTC-USD');
    expect(parseSymbolParam(encodeURIComponent('btc-usd'))).toBe('BTC-USD');
  });
});

describe('detail search params', () => {
  it('parses interval and limit with bounds', () => {
    const s = parseDetailSearchParams(new URLSearchParams('interval=4h&limit=200'));
    expect(s).toEqual({ interval: '4h', limit: 200 });
    expect(parseDetailSearchParams(new URLSearchParams('limit=5')).limit).toBe(100);
  });

  it('omits defaults when serializing', () => {
    const p = detailStateToSearchParams({ interval: '1h', limit: 100 });
    expect(p.toString()).toBe('');
  });
});

describe('resolveInterval', () => {
  it('keeps supported request and falls back otherwise', () => {
    expect(resolveInterval('15m', ['1m', '15m', '1h'])).toBe('15m');
    expect(resolveInterval('3m', ['1h', '1d'])).toBe('1h');
    expect(resolveInterval('3m', ['5m', '15m'])).toBe('5m');
  });
});
