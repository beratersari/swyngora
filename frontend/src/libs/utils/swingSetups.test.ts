import { describe, expect, it } from 'vitest';
import type { ScannerResult } from '@/libs/api';
import {
  backtestRangeIso,
  buildSwingSetups,
  countHitsSince,
  describeRule,
  gradeFromScore,
  ruleFactorsShort,
  ruleTypeShort,
} from './swingSetups';

function hit(partial: Partial<ScannerResult> & Pick<ScannerResult, 'ruleType'>): ScannerResult {
  return {
    id: partial.id ?? `${partial.ruleType}-${partial.marketDataKey ?? 'x'}`,
    ruleId: partial.ruleId ?? 'r1',
    exchange: partial.exchange ?? 'binance',
    symbol: partial.symbol ?? 'BTCUSDT',
    ruleType: partial.ruleType,
    interval: partial.interval ?? '4h',
    marketDataKey: partial.marketDataKey ?? '2026-08-01T00:00:00Z',
    matchedAt: partial.matchedAt ?? '2026-08-01T00:05:00Z',
    summary: partial.summary ?? `${partial.ruleType} hit`,
  };
}

describe('gradeFromScore', () => {
  it('maps 3/2/1 to A/B/C', () => {
    expect(gradeFromScore(3)).toBe('A');
    expect(gradeFromScore(2)).toBe('B');
    expect(gradeFromScore(1)).toBe('C');
  });
});

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
      conditions: ['rsi', 'volume_increase'] as const,
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

describe('buildSwingSetups', () => {
  const now = Date.parse('2026-08-01T12:00:00Z');

  it('requires at least two factor types', () => {
    const out = buildSwingSetups(
      [
        hit({ ruleType: 'rsi', matchedAt: '2026-08-01T10:00:00Z' }),
        hit({ id: 'rsi2', ruleType: 'rsi', matchedAt: '2026-08-01T11:00:00Z', marketDataKey: 'b2' }),
      ],
      now,
    );
    expect(out).toHaveLength(0);
  });

  it('grades same-pair confluence and flags same-bar overlap', () => {
    const bar = '2026-08-01T08:00:00Z';
    const out = buildSwingSetups(
      [
        hit({ ruleType: 'rsi', marketDataKey: bar, matchedAt: '2026-08-01T08:01:00Z' }),
        hit({ ruleType: 'ma_crossover', marketDataKey: bar, matchedAt: '2026-08-01T08:01:00Z' }),
        hit({
          ruleType: 'volume_increase',
          marketDataKey: '2026-08-01T04:00:00Z',
          matchedAt: '2026-08-01T04:01:00Z',
        }),
      ],
      now,
    );
    expect(out).toHaveLength(1);
    expect(out[0]?.score).toBe(3);
    expect(out[0]?.grade).toBe('A');
    expect(out[0]?.sameBar).toBe(true);
    expect(out[0]?.factors).toEqual(['ma_crossover', 'rsi', 'volume_increase']);
  });

  it('drops hits outside the window', () => {
    const out = buildSwingSetups(
      [
        hit({
          ruleType: 'rsi',
          matchedAt: '2026-07-01T00:00:00Z',
          marketDataKey: 'old',
        }),
        hit({
          ruleType: 'ma_crossover',
          matchedAt: '2026-08-01T10:00:00Z',
          marketDataKey: 'new',
        }),
      ],
      now,
    );
    expect(out).toHaveLength(0);
  });
});

describe('countHitsSince', () => {
  it('counts recent matches', () => {
    const since = Date.parse('2026-08-01T00:00:00Z');
    expect(
      countHitsSince(
        [
          hit({ ruleType: 'rsi', matchedAt: '2026-08-01T01:00:00Z' }),
          hit({ ruleType: 'rsi', id: 'old', matchedAt: '2026-07-01T01:00:00Z' }),
        ],
        since,
      ),
    ).toBe(1);
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
