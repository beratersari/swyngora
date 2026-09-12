import { describe, expect, it } from 'vitest';
import { formatArbMoney, formatArbPct, routeLabel } from './helpers';

describe('PriceDiffScanTable helpers', () => {
  it('formats money and percent', () => {
    expect(formatArbMoney(1234.56)).toContain('1');
    expect(formatArbPct(0.42)).toBe('0.420%');
    expect(formatArbMoney(undefined)).toBe('—');
  });

  it('labels a buy→sell route', () => {
    expect(routeLabel({ buyExchange: 'binance', sellExchange: 'bybit' })).toBe('binance → bybit');
  });
});
