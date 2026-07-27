import { describe, expect, it } from 'vitest';
import {
  DEFAULT_DETAIL_STATE,
  detailStateToSearchParams,
  marketsBackPath,
  parseDetailSearchParams,
  parseExchangeParam,
  parseSymbolParam,
  resolveInterval,
  toSupplyAsset,
} from './detailQuery';

describe('parseExchangeParam', () => {
  it('accepts known venues (case-insensitive) and defaults unknown', () => {
    expect(parseExchangeParam('coinbase')).toBe('coinbase');
    expect(parseExchangeParam('BINANCE')).toBe('binance');
    expect(parseExchangeParam('Bybit')).toBe('bybit');
    expect(parseExchangeParam('NOPE')).toBe('binance');
    expect(parseExchangeParam(undefined)).toBe('binance');
    expect(parseExchangeParam('')).toBe('binance');
  });
});

describe('parseSymbolParam', () => {
  it('decodes and uppercases', () => {
    expect(parseSymbolParam('btc-usd')).toBe('BTC-USD');
    expect(parseSymbolParam(encodeURIComponent('btc-usd'))).toBe('BTC-USD');
    expect(parseSymbolParam(undefined)).toBe('');
    expect(parseSymbolParam('  eth  ')).toBe('ETH');
  });

  it('falls back when decodeURIComponent throws', () => {
    // lone % can throw in some engines
    const bad = '%E0%A4%A';
    expect(parseSymbolParam(bad).length).toBeGreaterThan(0);
  });
});

describe('detail search params', () => {
  it('parses interval and limit with bounds', () => {
    const s = parseDetailSearchParams(new URLSearchParams('interval=4h&limit=200'));
    expect(s).toEqual({ interval: '4h', limit: 200 });
    expect(parseDetailSearchParams(new URLSearchParams('limit=5')).limit).toBe(100);
    expect(parseDetailSearchParams(new URLSearchParams('limit=501')).limit).toBe(100);
    // Floor then clamp: 500.1 → 500 (not default)
    expect(parseDetailSearchParams(new URLSearchParams('limit=500.1')).limit).toBe(500);
    expect(parseDetailSearchParams(new URLSearchParams('limit=19.9')).limit).toBe(100);
    expect(parseDetailSearchParams(new URLSearchParams('limit=abc')).limit).toBe(100);
    expect(parseDetailSearchParams(new URLSearchParams()).interval).toBe(
      DEFAULT_DETAIL_STATE.interval,
    );
  });

  it('omits defaults when serializing', () => {
    const p = detailStateToSearchParams({ interval: '1h', limit: 100 });
    expect(p.toString()).toBe('');
    const p2 = detailStateToSearchParams({ interval: '4h', limit: 200 });
    expect(p2.get('interval')).toBe('4h');
    expect(p2.get('limit')).toBe('200');
  });
});

describe('resolveInterval', () => {
  it('keeps supported request and falls back otherwise', () => {
    expect(resolveInterval('15m', ['1m', '15m', '1h'])).toBe('15m');
    expect(resolveInterval('3m', ['1h', '1d'])).toBe('1h');
    expect(resolveInterval('3m', ['5m', '15m'])).toBe('5m');
    expect(resolveInterval('3m', undefined)).toBe('3m');
    expect(resolveInterval('', undefined)).toBe(DEFAULT_DETAIL_STATE.interval);
    expect(resolveInterval('3m', [])).toBe('3m');
  });
});

describe('toSupplyAsset', () => {
  it('strips Coinbase hyphen quotes to base', () => {
    expect(toSupplyAsset('BTC-USD')).toBe('BTC');
    expect(toSupplyAsset('eth-usdt')).toBe('ETH');
    expect(toSupplyAsset('-USD')).toBe('-USD');
  });

  it('strips unhyphenated stable suffixes (longest first)', () => {
    expect(toSupplyAsset('BTCUSDT')).toBe('BTC');
    expect(toSupplyAsset('ETHUSDC')).toBe('ETH');
    expect(toSupplyAsset('XFDUSD')).toBe('X');
  });

  it('keeps bare base tickers and empty', () => {
    expect(toSupplyAsset('BTC')).toBe('BTC');
    expect(toSupplyAsset('RLUSD')).toBe('RLUSD');
    expect(toSupplyAsset('')).toBe('');
    expect(toSupplyAsset('   ')).toBe('');
  });
});

describe('marketsBackPath', () => {
  it('defaults binance to bare /markets', () => {
    expect(marketsBackPath('binance')).toBe('/markets');
    expect(marketsBackPath('BINANCE')).toBe('/markets');
    expect(marketsBackPath('')).toBe('/markets');
  });

  it('preserves non-default exchange', () => {
    expect(marketsBackPath('coinbase')).toBe('/markets?exchange=coinbase');
    expect(marketsBackPath('bybit')).toBe('/markets?exchange=bybit');
  });
});
