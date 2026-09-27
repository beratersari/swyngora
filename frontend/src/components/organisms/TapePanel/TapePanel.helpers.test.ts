import { describe, expect, it } from 'vitest';
import { findPerpCoin, formatFundingPct, formatSignedFundingPct, formatTapeNum, perpBidShareLine, perpDepthLine, windowByName } from './TapePanel.helpers';

describe('TapePanel helpers', () => {
  it('compacts large notionals', () => {
    expect(formatTapeNum(1_500_000_000)).toBe('1.50B');
    expect(formatTapeNum('')).toBe('—');
  });

  it('formats funding percents', () => {
    expect(formatFundingPct(0.0123)).toBe('0.0123%');
    expect(formatFundingPct(null)).toBe('—');
  });

  it('keeps the sign on funding pressure change', () => {
    expect(formatSignedFundingPct('+0.001')).toBe('+0.0010%');
    expect(formatSignedFundingPct('-0.002')).toBe('-0.0020%');
    expect(formatSignedFundingPct(null)).toBe('—');
  });

  it('picks a named window', () => {
    expect(windowByName([{ window: '1h' }, { window: '24h' }], '24h')?.window).toBe('24h');
  });

  it('formats bid share then to now', () => {
    expect(perpBidShareLine({ complete: true, pastBidPct: '55', bidPct: '65', changePct: '+10' })).toBe(
      '55% → 65% (+10)',
    );
    expect(
      perpDepthLine({ complete: true, pastTotal: '400000000', total: '500000000', totalChangePct: '+25' }),
    ).toBe('400.00M → 500.00M (+25%)');
    expect(perpDepthLine({ complete: true })).toBe('—');
  });

  it('finds this coin in the market perp book', () => {
    const coins = [
      { symbol: 'ETHUSDT', base: 'ETH' },
      { symbol: 'BTCUSDT', base: 'BTC' },
    ];
    expect(findPerpCoin(coins, 'BTCUSDT')?.base).toBe('BTC');
    expect(findPerpCoin(coins, 'ETH')?.base).toBe('ETH');
  });
});
