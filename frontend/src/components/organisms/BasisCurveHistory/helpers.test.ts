import { describe, expect, it } from 'vitest';
import { hasCurveHistory, toCurveLine, windowChangeLabel } from './helpers';

describe('BasisCurveHistory helpers', () => {
  it('builds a spread line inside the range', () => {
    const now = Date.parse('2026-09-19T16:00:00.000Z');
    const line = toCurveLine(
      [
        { time: '2026-09-19T10:00:00.000Z', spreadPct: '-0.20' },
        { time: '2026-09-19T15:00:00.000Z', spreadPct: '+0.40' },
        { time: '2026-09-10T16:00:00.000Z', spreadPct: '+1.00' },
      ],
      '24h',
      now,
    );
    expect(line).toHaveLength(2);
    expect(line[1]?.value).toBe(0.4);
    expect(
      hasCurveHistory(
        [
          { time: '2026-09-19T10:00:00.000Z', spreadPct: '0.1' },
          { time: '2026-09-19T11:00:00.000Z', spreadPct: '0.2' },
        ],
        '24h',
        now,
      ),
    ).toBe(true);
  });

  it('labels a flip window', () => {
    expect(
      windowChangeLabel({
        window: '4h',
        complete: true,
        change: 'flipped',
        pastShape: 'backwardation',
        shape: 'contango',
      }),
    ).toBe('backwardation → contango');
  });
});
