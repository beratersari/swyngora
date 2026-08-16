import { describe, expect, it } from 'vitest';
import {
  analyticsBarLimit,
  DEFAULT_DETAIL_STATE,
  detailStateToSearchParams,
  intervalToSeconds,
  marketsBackPath,
  parseDetailSearchParams,
  parseExchangeParam,
  parseSymbolParam,
  resolveInterval,
  toSupplyAsset,
} from './detailQuery';
describe('parseExchangeParam', () => {
  it('accepts known venues (case-insensitive) and rejects unknown', () => {
    expect(parseExchangeParam('coinbase')).toBe('coinbase');
    expect(parseExchangeParam('BINANCE')).toBe('binance');
    expect(parseExchangeParam('Bybit')).toBe('bybit');
    expect(parseExchangeParam('NOPE')).toBeNull();
    expect(parseExchangeParam(undefined)).toBeNull();
    expect(parseExchangeParam('')).toBeNull();
    expect(parseExchangeParam('  ')).toBeNull();
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
    expect(s).toEqual({ interval: '4h', limit: 200, tab: 'overview' });
    expect(parseDetailSearchParams(new URLSearchParams('tab=holders')).tab).toBe('holders');
    expect(parseDetailSearchParams(new URLSearchParams('tab=nope')).tab).toBe('overview');
    expect(parseDetailSearchParams(new URLSearchParams('limit=5')).limit).toBe(
      DEFAULT_DETAIL_STATE.limit,
    );
    // Out of range falls back to default live window
    expect(parseDetailSearchParams(new URLSearchParams('limit=1001')).limit).toBe(
      DEFAULT_DETAIL_STATE.limit,
    );
    expect(parseDetailSearchParams(new URLSearchParams('limit=500.1')).limit).toBe(500);
    expect(parseDetailSearchParams(new URLSearchParams('limit=19.9')).limit).toBe(
      DEFAULT_DETAIL_STATE.limit,
    );
    expect(parseDetailSearchParams(new URLSearchParams('limit=abc')).limit).toBe(
      DEFAULT_DETAIL_STATE.limit,
    );
    expect(parseDetailSearchParams(new URLSearchParams()).interval).toBe(
      DEFAULT_DETAIL_STATE.interval,
    );
  });

  it('serializes interval only (limit is progressive / not in URL)', () => {
    const p = detailStateToSearchParams({ interval: '1h', limit: 100 });
    expect(p.toString()).toBe('');
    const p2 = detailStateToSearchParams({ interval: '4h', limit: 200 });
    expect(p2.get('interval')).toBe('4h');
    expect(p2.get('limit')).toBeNull();
    const p3 = detailStateToSearchParams({ interval: '1h', tab: 'holders' });
    expect(p3.get('tab')).toBe('holders');
    expect(p3.get('interval')).toBeNull();
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

describe('intervalToSeconds', () => {
  it('parses m/h/d/w', () => {
    expect(intervalToSeconds('15m')).toBe(900);
    expect(intervalToSeconds('1h')).toBe(3600);
    expect(intervalToSeconds('4h')).toBe(14400);
    expect(intervalToSeconds('1d')).toBe(86400);
    expect(intervalToSeconds('1w')).toBe(604800);
  });

  it('returns 0 for invalid', () => {
    expect(intervalToSeconds('')).toBe(0);
    expect(intervalToSeconds('1M')).toBe(0);
    expect(intervalToSeconds('foo')).toBe(0);
  });
});

describe('analyticsBarLimit', () => {
  it('floors at minLive and caps at apiMax', () => {
    expect(analyticsBarLimit(0, 100)).toBe(100);
    expect(analyticsBarLimit(50, 100)).toBe(100);
    expect(analyticsBarLimit(250, 100)).toBe(250);
    expect(analyticsBarLimit(5000, 100, 1000)).toBe(1000);
  });
});
