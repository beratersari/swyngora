import { describe, expect, it } from 'vitest';
import {
  asksHighToLow,
  depthPct,
  formatBookAmount,
  formatBookPrice,
  formatGroupLabel,
  groupLabelDecimals,
  maxNotional,
  priceDecimalsFromGroup,
  qtyDecimalsFromLevels,
  markdownBookRow,
  markdownRule,
  sharedPriceExponent,
  visibleGroupSteps,
} from './helpers';

describe('OrderBook helpers', () => {
  it('maxNotional and depthPct', () => {
    const max = maxNotional([
      { notional: '10' },
      { notional: '40' },
      { notional: 'x' },
    ]);
    expect(max).toBe(40);
    expect(depthPct('20', 40)).toBe(50);
    expect(depthPct('0', 40)).toBe(0);
  });

  it('asksHighToLow reverses best-first asks', () => {
    const out = asksHighToLow([{ price: '100.1' }, { price: '100.2' }]);
    expect(out.map((r) => r.price)).toEqual(['100.2', '100.1']);
  });

  it('priceDecimalsFromGroup follows the group step', () => {
    expect(priceDecimalsFromGroup('1')).toBe(0);
    expect(priceDecimalsFromGroup('0.1')).toBe(1);
    expect(priceDecimalsFromGroup('0.10')).toBe(1);
    expect(priceDecimalsFromGroup('0.01')).toBe(2);
    expect(priceDecimalsFromGroup('0.00001')).toBe(5);
    expect(priceDecimalsFromGroup(undefined)).toBe(2);
  });

  it('qtyDecimalsFromLevels uses the widest visible fraction', () => {
    expect(
      qtyDecimalsFromLevels([
        { quantity: '2', cumulative: '2' },
        { quantity: '1.25', cumulative: '3.25' },
      ]),
    ).toBe(2);
  });

  it('formatBookAmount pads to a fixed scale', () => {
    expect(formatBookAmount('100', 1)).toBe('100.0');
    expect(formatBookAmount('99.5', 1)).toBe('99.5');
    expect(formatBookAmount('1.2', 4)).toBe('1.2000');
    expect(formatBookAmount('x', 2)).toBe('—');
  });

  it('formatGroupLabel pads every step to the same decimal column', () => {
    const steps = ['0.000001', '0.00001', '0.0001'];
    const decimals = groupLabelDecimals(steps);
    expect(decimals).toBe(6);
    expect(formatGroupLabel('0.000001', decimals)).toBe('0.000001');
    expect(formatGroupLabel('0.00001', decimals)).toBe('0.000010');
    expect(formatGroupLabel('0.0001', decimals)).toBe('0.000100');
    expect(formatGroupLabel('0.1')).toBe('0.1');
  });

  it('formatBookPrice uses a shared exponent for tiny books', () => {
    expect(sharedPriceExponent('0.00000001', '0.00001234')).toBe(-5);
    expect(formatBookPrice('0.00001234', 8, -5)).toBe('1.23e-5');
    expect(formatBookPrice('0.00001200', 8, -5)).toBe('1.20e-5');
    expect(formatBookPrice('100.1', 1, -5)).toBe('100.1');
    expect(formatBookPrice('100.1', 1, null)).toBe('100.1');
  });

  it('visibleGroupSteps drops a stale finer group', () => {
    expect(visibleGroupSteps(['0.000001', '0.00001', '0.0001'], '0.00000001')).toEqual([
      '0.000001',
      '0.00001',
      '0.0001',
    ]);
    expect(visibleGroupSteps(['0.000001', '0.00001'], '0.000005')).toEqual([
      '0.000005',
      '0.000001',
      '0.00001',
    ]);
  });

  it('markdownBookRow pads cells like a pipe table', () => {
    const widths = [8, 4];
    expect(markdownBookRow(['0.000001', '12'], widths)).toBe('| 0.000001 |   12 |');
    expect(markdownRule(widths)).toBe('| -------: | ---: |');
  });
});
