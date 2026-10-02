import { describe, expect, it } from 'vitest';
import { coinChartPath, definiteText, fundingText, signedAmount } from './helpers';
import type { DerivativesCoin } from '@/libs/api';

describe('derivatives board helpers', () => {
  it('opens the binance chart for a combined row', () => {
    expect(coinChartPath('all', 'BTCUSDT')).toBe('/markets/binance/BTCUSDT');
    expect(coinChartPath('bybit', 'ETHUSDT')).toBe('/markets/bybit/ETHUSDT');
  });

  it('withholds a combined number that is not definite', () => {
    expect(definiteText(false, '100', (value) => value)).toBe('—');
    expect(definiteText(true, '100', (value) => value)).toBe('100');
  });

  it('signs a compact open-interest change', () => {
    expect(signedAmount('1500000000')).toBe('+1.50B');
    expect(signedAmount('-2500000')).toBe('-2.50M');
  });

  it('leaves funding blank until every venue is fresh', () => {
    const coin = { fundingDefinite: false, fundingRatePct: '+0.01' } as DerivativesCoin;
    expect(fundingText(coin)).toBe('—');
    expect(fundingText({ ...coin, fundingDefinite: true })).toBe('+0.01%');
  });
});
