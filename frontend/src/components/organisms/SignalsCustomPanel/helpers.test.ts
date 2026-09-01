import { describe, expect, it } from 'vitest';
import type { CustomSignalItem } from '@/libs/api';
import { defaultCustomWeightDraft, previewCustomMix, customWeightsValid, weightsFromDraft } from './helpers';

function item(partial: Partial<CustomSignalItem> & { symbol: string }): CustomSignalItem {
  return {
    exchange: 'binance',
    custom: {
      score: 50,
      grade: 'C',
      factors: [
        { id: 'ema_crossover', score: 90, status: 'used', requestedPct: 16, sharePct: 16, effect: 6.4 },
        { id: 'hunt_up', score: 80, status: 'used', requestedPct: 0, sharePct: 0, effect: 0 },
      ],
    },
    scoreCompare: { default: { score: 50 }, applied: { score: 50 }, delta: { score: 0 } },
    ...partial,
  };
}

describe('custom signal helpers', () => {
  it('built-in draft sums to 100 and is valid', () => {
    const draft = defaultCustomWeightDraft();
    expect(customWeightsValid(draft)).toBe(true);
    expect(weightsFromDraft(draft).hunt_up).toBe(0);
  });

  it('previews score change and names the largest mover', () => {
    const draft = defaultCustomWeightDraft().map((row) =>
      row.id === 'hunt_up'
        ? { ...row, enabled: true, pct: 100 }
        : { ...row, enabled: false, pct: 0 },
    );
    const rows = previewCustomMix([item({ symbol: 'ETHUSDT' })], draft);
    expect(rows[0]?.appliedScore).toBe(80);
    expect(rows[0]?.largestChange?.id).toBe('hunt_up');
    expect(rows[0]?.factors.find((f) => f.id === 'hunt_up')?.appliedEffect).toBe(30);
  });

  it('missing hunt keeps its share and adds 0', () => {
    const draft = defaultCustomWeightDraft().map((row) =>
      row.id === 'hunt_up'
        ? { ...row, enabled: true, pct: 50 }
        : row.id === 'ema_crossover'
          ? { ...row, enabled: true, pct: 50 }
          : { ...row, enabled: false, pct: 0 },
    );
    const rows = previewCustomMix(
      [
        item({
          symbol: 'BTCUSDT',
          custom: {
            score: 50,
            factors: [
              { id: 'ema_crossover', score: 90, status: 'used' },
              { id: 'hunt_up', score: 50, status: 'missing' },
            ],
          },
        }),
      ],
      draft,
    );
    expect(rows[0]?.appliedScore).toBe(70);
    expect(rows[0]?.huntMissing).toBe(true);
    expect(rows[0]?.factors.find((f) => f.id === 'hunt_up')?.appliedEffect).toBe(0);
  });
});
