import { describe, expect, it } from 'vitest';
import { formatOppDuration, sparklinePoints } from './helpers';

describe('price-diff opportunity helpers', () => {
  it('formats duration', () => {
    expect(formatOppDuration(12)).toBe('12s');
    expect(formatOppDuration(90)).toBe('1m 30s');
    expect(formatOppDuration(3600)).toBe('1h');
  });

  it('builds a sparkline from ticks', () => {
    const pts = sparklinePoints([
      { at: '2026-09-13T12:00:00Z', netDiffPct: 0.4 },
      { at: '2026-09-13T12:01:00Z', netDiffPct: 0.8 },
      { at: '2026-09-13T12:02:00Z', netDiffPct: 0.2 },
    ]);
    expect(pts.split(' ')).toHaveLength(3);
    expect(sparklinePoints([{ netDiffPct: 0.4 }])).toBe('');
  });
});
