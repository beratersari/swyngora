import { describe, expect, it } from 'vitest';
import { windowMix } from './helpers';

describe('BreadthOverview helpers', () => {
  it('formats up vs down share', () => {
    expect(windowMix({ counts: { upPct: '62', downPct: '30' } })).toBe('62% up · 30% down');
  });
});
