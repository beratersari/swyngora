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
    expect(formatChangePercent(0)).toBe('0.00%');
    expect(formatChangePercent(null)).toBe('—');
    expect(formatChangePercent(undefined)).toBe('—');
    expect(formatChangePercent('')).toBe('—');
    expect(formatChangePercent('nope')).toBe('—');
  });

  it('maps change tone', () => {
    expect(changeTone(1)).toBe('success');
    expect(changeTone(-1)).toBe('error');
    expect(changeTone(0)).toBe('secondary');
    expect(changeTone(null)).toBe('secondary');
    expect(changeTone('')).toBe('secondary');
    expect(changeTone(Number.NaN)).toBe('secondary');
  });

  it('formats compact usd tiers', () => {
    expect(formatCompactUsd(1_500_000_000_000)).toBe('1.50T');
    expect(formatCompactUsd(1_500_000_000)).toBe('1.50B');
    expect(formatCompactUsd(2_500_000)).toBe('2.50M');
    expect(formatCompactUsd(3_200)).toBe('3.20K');
    expect(formatCompactUsd(0)).toBe('0');
    expect(formatCompactUsd(-1_500_000)).toBe('-1.50M');
    expect(formatCompactUsd('∞')).toBe('∞');
    expect(formatCompactUsd(null)).toBe('—');
    expect(formatCompactUsd('')).toBe('—');
    expect(formatCompactUsd(Number.NaN)).toBe('—');
    expect(formatCompactUsd(42)).toBe(formatCompactUsd(42));
  });

  it('hides zero trade count off binance and rejects non-finite', () => {
    expect(formatTradeCount(0, 'coinbase')).toBe('—');
    expect(formatTradeCount(0, 'binance')).toBe('0');
    expect(formatTradeCount(0, undefined)).toBe('0');
    expect(formatTradeCount(12, 'bybit')).toBe('12');
    expect(formatTradeCount(null, 'binance')).toBe('—');
    expect(formatTradeCount(Number.NaN, 'binance')).toBe('—');
    expect(formatTradeCount(Number.POSITIVE_INFINITY, 'binance')).toBe('—');
  });

  it('formats infinite max mcap and finite via compact', () => {
    expect(formatMarketCapMax('∞')).toBe('∞');
    expect(formatMarketCapMax(1_000_000)).toBe('1.00M');
  });
});
