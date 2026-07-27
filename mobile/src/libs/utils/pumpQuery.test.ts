import { describe, expect, it } from 'vitest';
import {
  buildDetailPumpQuery,
  buildScanQuery,
  defaultPumpScanFilters,
  defaultQuoteForExchange,
} from './pumpQuery';

describe('defaultQuoteForExchange', () => {
  it('USD on coinbase else USDT', () => {
    expect(defaultQuoteForExchange('coinbase')).toBe('USD');
    expect(defaultQuoteForExchange('binance')).toBe('USDT');
  });
});

describe('defaultPumpScanFilters', () => {
  it('uses service-aligned defaults', () => {
    const f = defaultPumpScanFilters('binance');
    expect(f.interval).toBe('15m');
    expect(f.lookbackHours).toBe(24);
    expect(f.minReturnPct).toBe(8);
    expect(f.direction).toBe('up');
    expect(f.symbolLimit).toBe(15);
  });
});

describe('buildScanQuery / buildDetailPumpQuery', () => {
  it('passes through scan fields', () => {
    const f = defaultPumpScanFilters();
    expect(buildScanQuery(f).quote).toBe('USDT');
  });

  it('builds detail query', () => {
    const q = buildDetailPumpQuery({
      exchange: 'binance',
      symbol: 'BTCUSDT',
      interval: '1h',
      lookbackHours: 48,
      minReturnPct: 5,
      direction: 'both',
      maxEvents: 10,
    });
    expect(q.symbol).toBe('BTCUSDT');
    expect(q.direction).toBe('both');
  });
});
