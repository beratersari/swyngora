import { describe, expect, it } from 'vitest';
import {
  buildBatchIndicatorsArg,
  buildBatchIndicatorsBody,
  chunkSymbols,
  formatRsi,
  groupPairsByExchange,
  indexBatchItemsBySymbol,
  rsiFieldsFromItem,
  rsiTone,
} from './batchIndicators';

describe('chunkSymbols', () => {
  it('chunks by max size', () => {
    expect(chunkSymbols(['a', 'b', 'c', 'd'], 2)).toEqual([
      ['a', 'b'],
      ['c', 'd'],
    ]);
    expect(chunkSymbols([], 50)).toEqual([]);
  });
});

describe('groupPairsByExchange', () => {
  it('groups unique symbols per venue', () => {
    const g = groupPairsByExchange([
      { exchange: 'binance', symbol: 'btcusdt' },
      { exchange: 'BINANCE', symbol: 'BTCUSDT' },
      { exchange: 'coinbase', symbol: 'ETH-USD' },
      { exchange: 'bybit', symbol: 'SOLUSDT' },
    ]);
    expect(g.binance).toEqual(['BTCUSDT']);
    expect(g.coinbase).toEqual(['ETH-USD']);
    expect(g.bybit).toEqual(['SOLUSDT']);
  });
});

describe('buildBatchIndicatorsArg', () => {
  it('normalizes, dedupes, caps, and fills defaults', () => {
    const arg = buildBatchIndicatorsArg({
      exchange: 'Bybit',
      symbols: ['btcusdt', 'BTCUSDT', 'ethusdt', ''],
      maxSymbols: 2,
    });
    expect(arg.exchange).toBe('bybit');
    expect(arg.symbols).toEqual(['BTCUSDT', 'ETHUSDT']);
    expect(arg.interval).toBe('1h');
    expect(arg.rsiPeriod).toBe(14);
    expect(arg.emaPeriods).toBe('12,26');
  });
});

describe('buildBatchIndicatorsBody', () => {
  it('serializes body fields', () => {
    expect(
      buildBatchIndicatorsBody({
        exchange: 'binance',
        symbols: ['BTCUSDT'],
        interval: '4h',
        rsiPeriod: 14,
        emaPeriods: '12,26',
      }),
    ).toEqual({
      exchange: 'binance',
      interval: '4h',
      symbols: ['BTCUSDT'],
      rsiPeriod: 14,
      emaPeriods: '12,26',
    });
  });
});

describe('indexBatchItemsBySymbol', () => {
  it('indexes by uppercase symbol', () => {
    const map = indexBatchItemsBySymbol([
      { symbol: 'btcUsdt', rsi: 55 },
      { symbol: 'ETHUSDT', error: 'unavailable' },
    ]);
    expect(map.get('BTCUSDT')?.rsi).toBe(55);
    expect(map.get('ETHUSDT')?.error).toBe('unavailable');
  });
});

describe('formatRsi / rsiTone / rsiFieldsFromItem', () => {
  it('formats and tones', () => {
    expect(formatRsi(62.456)).toBe('RSI 62.5');
    expect(formatRsi(null)).toBe('—');
    expect(rsiTone(75)).toBe('warning');
    expect(rsiTone(25)).toBe('success');
    expect(rsiTone(50)).toBe('secondary');
  });

  it('maps item to row fields', () => {
    expect(rsiFieldsFromItem(undefined, true)).toEqual({
      rsiLabel: '…',
      rsiTone: 'secondary',
      rsiLoading: true,
    });
    expect(rsiFieldsFromItem({ symbol: 'X', error: 'unavailable' }, false).rsiLabel).toBe('—');
    expect(rsiFieldsFromItem({ symbol: 'X', rsi: 40 }, false)).toEqual({
      rsiLabel: 'RSI 40.0',
      rsiTone: 'secondary',
      rsiLoading: false,
    });
  });
});
