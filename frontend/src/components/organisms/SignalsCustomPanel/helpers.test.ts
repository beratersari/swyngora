import { describe, expect, it } from 'vitest';
import type { CustomSignalItem } from '@/libs/api';
import {
  defaultCustomWeightDraft,
  previewCustomMix,
  customWeightsValid,
  weightsFromDraft,
  previewEvidence,
} from './helpers';

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
    expect(weightsFromDraft(draft).forensics).toBe(0);
    expect(weightsFromDraft(draft).similar_setup).toBe(0);
  });

  it('marks similar setups against a long as used when weighted', () => {
    const draft = defaultCustomWeightDraft().map((row) =>
      row.id === 'similar_setup'
        ? { ...row, enabled: true, pct: 100 }
        : { ...row, enabled: false, pct: 0 },
    );
    const rows = previewCustomMix(
      [
        item({
          symbol: 'SOLUSDT',
          custom: {
            score: 20,
            factors: [{ id: 'similar_setup', score: 20, status: 'used', detail: 'goes against a long custom signal' }],
          },
        }),
      ],
      draft,
    );
    expect(rows[0]?.factors.find((f) => f.id === 'similar_setup')?.appliedPct).toBe(100);
    expect(rows[0]?.factors.find((f) => f.id === 'similar_setup')?.appliedEffect).toBeLessThan(0);
    expect(rows[0]?.tapeMissing).toBe(false);
  });

  it('passes through how many similar setups were used vs missing tape', () => {
    const draft = defaultCustomWeightDraft();
    const rows = previewCustomMix(
      [
        item({
          symbol: 'BTCUSDT',
          tapeSetups: 5,
          tapeUsed: 3,
          tapeThin: 2,
          tapeFresh: { note: 'oi 6m ago; options 40m ago (old)', options: { stale: true } },
        }),
      ],
      draft,
    );
    expect(rows[0]?.tapeSetups).toBe(5);
    expect(rows[0]?.tapeUsed).toBe(3);
    expect(rows[0]?.tapeThin).toBe(2);
    expect(rows[0]?.tapeFreshNote).toBe('oi 6m ago; options 40m ago (old)');
    expect(rows[0]?.tapeFreshStale).toBe(true);
  });

  it('marks a high score as weak evidence when half the mix is missing tape', () => {
    const draft = defaultCustomWeightDraft().map((row) =>
      row.id === 'ema_crossover'
        ? { ...row, enabled: true, pct: 50 }
        : row.id === 'similar_setup'
          ? { ...row, enabled: true, pct: 50 }
          : { ...row, enabled: false, pct: 0 },
    );
    const rows = previewCustomMix(
      [
        item({
          symbol: 'SOLUSDT',
          custom: {
            score: 88,
            factors: [
              { id: 'ema_crossover', score: 90, status: 'used', quality: 100 },
              { id: 'similar_setup', score: 80, status: 'missing' },
            ],
          },
        }),
      ],
      draft,
    );
    expect(rows[0]?.appliedScore).toBeGreaterThan(70);
    expect(rows[0]?.evidenceLabel).toBe('fair');
    expect(rows[0]?.evidenceScore).toBeLessThanOrEqual(50);
  });

  it('scores evidence from used-factor quality', () => {
    const ev = previewEvidence([
      { id: 'ema_crossover', score: 90, status: 'used', defaultPct: 0, appliedPct: 40, defaultEffect: 0, appliedEffect: 0, deltaEffect: 0, quality: 100 },
      { id: 'similar_setup', score: 80, status: 'used', defaultPct: 0, appliedPct: 60, defaultEffect: 0, appliedEffect: 0, deltaEffect: 0, quality: 30 },
    ]);
    expect(ev.label).not.toBe('strong');
    expect(ev.score).toBe(58);
  });

  it('previews score change and names the largest mover', () => {
    const draft = defaultCustomWeightDraft().map((row) =>
      row.id === 'hunt_up'
        ? { ...row, enabled: true, pct: 100 }
        : { ...row, enabled: false, pct: 0 },
    );
    const rows = previewCustomMix([item({ symbol: 'ETHUSDT' })], draft);
    // One present factor: 80 * (0.9 + 0.1/5) = 73.6
    expect(rows[0]?.appliedScore).toBe(73.6);
    expect(rows[0]?.factors.find((f) => f.id === 'hunt_up')?.appliedPct).toBe(100);
  });

  it('does not score an idle pattern as 0', () => {
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
          engine: { swingScore: 61.5 },
          custom: {
            score: 50,
            factors: [
              { id: 'ema_crossover', score: 0, status: 'idle' },
              { id: 'hunt_up', score: 80, status: 'used' },
            ],
          },
        }),
      ],
      draft,
    );
    expect(rows[0]?.appliedScore).toBe(73.6);
    expect(rows[0]?.defaultScore).toBe(61.5);
    expect(rows[0]?.factors.find((f) => f.id === 'ema_crossover')?.status).toBe('idle');
    expect(rows[0]?.factors.find((f) => f.id === 'ema_crossover')?.appliedEffect).toBe(0);
  });
});
