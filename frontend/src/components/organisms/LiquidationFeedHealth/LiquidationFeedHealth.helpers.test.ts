import { describe, expect, it } from 'vitest';
import { formatClock, gapHours } from './LiquidationFeedHealth.helpers';

describe('LiquidationFeedHealth helpers', () => {
  it('formats clocks and sums gap hours', () => {
    expect(formatClock('2026-08-30T15:04:05.000Z')).toBe('15:04:05Z');
    expect(formatClock(undefined)).toBe('—');
    expect(gapHours({ gaps: [{ seconds: 3600 }, { seconds: 1800 }] })).toBe(1.5);
    expect(gapHours({ missingSeconds: 900, gaps: [{ seconds: 3600 }] })).toBe(0.25);
  });
});
