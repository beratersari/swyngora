import { describe, expect, it } from 'vitest';
import {
  changeTone,
  formatChangePercent,
  formatCompactUsd,
  formatMarketCapMax,
  formatTradeCount,
} from './formatMarket';

describe('formatMarket', () => {
  it('formats change percent with sign', () => {
    expect(formatChangePercent(1.5)).toBe('+1.50%');
    expect(formatChangePercent(-2)).toBe('-2.00%');
    expect(formatChangePercent(null)).toBe('—');
  });

  it('maps change tone', () => {
    expect(changeTone(1)).toBe('success');
    expect(changeTone(-1)).toBe('error');
    expect(changeTone(0)).toBe('secondary');
  });

  it('formats compact usd', () => {
    expect(formatCompactUsd(1_500_000_000)).toBe('1.50B');
    expect(formatCompactUsd(null)).toBe('—');
  });

  it('hides zero trade count off binance', () => {
    expect(formatTradeCount(0, 'coinbase')).toBe('—');
    expect(formatTradeCount(0, 'binance')).toBe('0');
    expect(formatTradeCount(12, 'bybit')).toBe('12');
  });

  it('formats infinite max mcap', () => {
    expect(formatMarketCapMax('∞')).toBe('∞');
  });
});
