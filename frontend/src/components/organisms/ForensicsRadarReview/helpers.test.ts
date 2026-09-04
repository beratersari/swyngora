import { describe, expect, it } from 'vitest';
import { radarAfterCell, radarHorizonLine } from './helpers';

describe('ForensicsRadarReview helpers', () => {
  it('formats filled and pending after cells', () => {
    expect(radarAfterCell({ after: [{ horizon: '1h' }] }, '1h')).toBe('…');
    expect(
      radarAfterCell(
        { after: [{ horizon: '1h', filledAt: '2026-09-04T13:00:00Z', returnPct: '+1.2', direction: 'up', useful: true }] },
        '1h',
      ),
    ).toBe('+1.2 ↑ ✓');
    expect(radarAfterCell({}, '4h')).toBe('—');
  });

  it('formats horizon usefulness', () => {
    expect(radarHorizonLine('1h', '67', 12)).toBe('1h: 67% useful (n=12)');
    expect(radarHorizonLine('15m')).toBe('15m: —');
  });
});
