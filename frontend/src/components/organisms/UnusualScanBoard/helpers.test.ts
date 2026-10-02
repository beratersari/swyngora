import { describe, expect, it } from 'vitest';
import { unusualSpanKey, unusualTapePath, unusualValue } from './helpers';

describe('unusual scan helpers', () => {
  it('opens Bybit on its own tape and OKX on the Binance tape', () => {
    expect(unusualTapePath('bybit', 'ETHUSDT')).toBe('/markets/bybit/ETHUSDT?tab=tape');
    expect(unusualTapePath('okx', 'BTCUSDT')).toBe('/markets/binance/BTCUSDT?tab=tape');
    expect(unusualTapePath('binance', 'SOLUSDT')).toBe('/markets/binance/SOLUSDT?tab=tape');
  });

  it('formats the current figure from the series unit', () => {
    expect(unusualValue({ unit: 'rate', current: 0.0001 })).toBe('0.0100%');
    expect(unusualValue({ unit: 'pct_change', current: 12.5 })).toBe('+12.50%');
    expect(unusualValue({ unit: 'signed_notional', current: -1500 })).toBe('-$1.50K');
  });

  it('labels the stored span in hours or days', () => {
    expect(unusualSpanKey(0.2)).toEqual({ key: 'span.underHour' });
    expect(unusualSpanKey(8)).toEqual({ key: 'span.hours', count: 8 });
    expect(unusualSpanKey(72)).toEqual({ key: 'span.days', count: 3 });
  });
});
