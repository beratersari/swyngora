import { describe, expect, it } from 'vitest';
import type { OrderPreview } from '@/libs/api';
import { formatPreviewPct, previewRows, previewWarnings } from './helpers';

const buy: OrderPreview = {
  product: 'spot',
  type: 'market',
  side: 'buy',
  symbol: 'BTCUSDT',
  fillPrice: 100,
  fee: 1,
  cashDebit: 101,
  cashAfter: 899,
  availableAfter: 899,
  fillsNow: true,
  risk: {
    applies: true,
    blocked: true,
    reasons: ['BTC would be 40.00% of the portfolio (limit 30.00%)'],
    asset: 'BTC',
    weightLimitHit: true,
    projectedWeightPct: 40,
    dailyLossLimitHit: false,
  },
  legs: [],
};

describe('previewRows', () => {
  it('lists fill, fee, and cash out for a buy', () => {
    const rows = previewRows(buy);
    expect(rows.find((row) => row.id === 'fee')?.value).toBe('1');
    expect(rows.find((row) => row.id === 'fill')?.value).toBe('100');
    expect(rows.find((row) => row.id === 'debit')?.value).toBe('101');
    expect(rows.find((row) => row.id === 'cashAfter')?.value).toBe('899');
  });
});

describe('previewWarnings', () => {
  it('names a blocked buy and a weight limit', () => {
    expect(previewWarnings(buy)).toEqual(['blocked', 'weight']);
  });

  it('keeps a daily-loss hit on a sell that is not blocked', () => {
    const sell: OrderPreview = {
      ...buy,
      side: 'sell',
      cashDebit: 0,
      cashCredit: 99,
      risk: { applies: false, blocked: false, dailyLossLimitHit: true, reasons: [] },
    };
    expect(previewWarnings(sell)).toEqual(['dailyLoss']);
  });
});

describe('formatPreviewPct', () => {
  it('rounds to one decimal without grouping', () => {
    expect(formatPreviewPct(40)).toBe('40%');
    expect(formatPreviewPct(12.54)).toBe('12.5%');
  });
});
