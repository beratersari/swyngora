import { describe, expect, it } from 'vitest';
import type { ScannerCondition } from '@/libs/api';
import { backtestRangeIso, describeRule, ruleFactorsShort, ruleTypeShort } from './swingSetups';

describe('ruleTypeShort / describeRule', () => {
  it('labels known types', () => {
    expect(ruleTypeShort('rsi')).toBe('RSI');
    expect(ruleTypeShort('ma_crossover')).toBe('EMA');
    expect(ruleTypeShort('volume_increase')).toBe('VOL');
    expect(ruleTypeShort('combo')).toBe('COMBO');
  });

  it('describes rule params', () => {
    expect(
      describeRule({
        id: '1',
        type: 'rsi',
        interval: '4h',
        enabled: true,
        rsiPeriod: 14,
        rsiCondition: 'below',
        rsiThreshold: 40,
      }),
    ).toContain('RSI(14)');
  });

  it('joins combo conditions with match mode', () => {
    const combo = {
      id: '2',
      type: 'combo' as const,
      conditions: ['rsi', 'volume_increase'] as ScannerCondition[],
      matchMode: 'any' as const,
      interval: '4h',
      enabled: true,
      rsiPeriod: 14,
      rsiCondition: 'below' as const,
      rsiThreshold: 40,
      volumeMinRatio: 2,
      volumeLookback: 20,
    };
    expect(describeRule(combo)).toContain(' or ');
    expect(ruleFactorsShort(combo)).toBe('RSI+VOL');
    expect(describeRule({ ...combo, matchMode: 'all' })).toContain(' and ');
  });
});

describe('backtestRangeIso', () => {
  it('returns RFC3339 span of N days', () => {
    const end = new Date('2026-08-01T00:00:00.000Z');
    const { rangeStart, rangeEnd } = backtestRangeIso(90, end);
    expect(rangeEnd).toBe('2026-08-01T00:00:00.000Z');
    expect(rangeStart).toBe('2026-05-03T00:00:00.000Z');
  });
});
