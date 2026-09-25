import { describe, expect, it } from 'vitest';
import { changeBarPct, maxAbsChange, signedChange, windowRows } from './helpers';

const expiry = '2026-03-27T08:00:00Z';

describe('windowRows', () => {
  it('keeps the selected expiry on a complete window', () => {
    const rows = windowRows(
      [
        {
          window: '4h',
          complete: true,
          strikes: [
            { expiry, strike: '100000', callChange: '6' },
            { expiry: '2026-04-24T08:00:00Z', strike: '100000', callChange: '1' },
          ],
        },
        { window: '1h', complete: false, strikes: [{ expiry, strike: '1' }] },
      ],
      '4h',
      expiry,
    );
    expect(rows).toHaveLength(1);
    expect(rows[0]?.strike).toBe('100000');
  });

  it('returns nothing while the window is still collecting', () => {
    expect(windowRows([{ window: '24h', complete: false, strikes: [] }], '24h', expiry)).toEqual([]);
  });
});

describe('change bars', () => {
  it('scales to the largest call or put move', () => {
    const peak = maxAbsChange([
      { callChange: '10', putChange: '-4' },
      { callChange: '2', putChange: '-20' },
    ]);
    expect(peak).toBe(20);
    expect(changeBarPct('-20', peak)).toBe(100);
    expect(changeBarPct('0', peak)).toBe(0);
    expect(signedChange('-3.5')).toBe(-3.5);
  });
});
