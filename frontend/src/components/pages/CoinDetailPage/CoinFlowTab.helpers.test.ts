import { describe, expect, it } from 'vitest';
import { perpStats, reportSummary, surgeStats } from './CoinFlowTab.helpers';

describe('CoinFlowTab helpers', () => {
  it('prefers top-level summary', () => {
    expect(reportSummary({ summary: 'Buy flow', venues: [{ summary: 'other' }] })).toBe('Buy flow');
  });

  it('reads perp headline notionals', () => {
    const stats = perpStats({
      combined: { headline: { below: { notional: '40000000' }, above: { notional: '15000000' }, lean: 'bid' } },
    });
    expect(stats.some((s) => s.label === 'Lean' && s.value === 'bid')).toBe(true);
  });

  it('reads 15m surge ratio', () => {
    const stats = surgeStats({
      venues: [{ windows: [{ window: '15m', ratio: '3.2', grade: 'high' }] }],
    });
    expect(stats).toEqual([
      { label: '15m', value: '3.2×' },
      { label: 'Grade', value: 'high' },
    ]);
  });
});
