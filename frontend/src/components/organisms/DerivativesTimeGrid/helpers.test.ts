import { describe, expect, it } from 'vitest';
import { cvdScale, gridColumnLabel, gridDetailVenues, paintGridCell } from './helpers';

describe('paintGridCell', () => {
  it('keeps a missing venue blank and does not paint zero', () => {
    const painted = paintGridCell(
      'funding',
      { definite: false, missing: ['okx'], value: 0, text: '0%' },
      1,
      'blank',
      (venues) => `missing ${venues}`,
    );
    expect(painted.tone).toBe('blank');
    expect(painted.text).toBe('—');
    expect(painted.title).toBe('missing okx');
  });

  it('colors funding by the rate and liquidations by heat', () => {
    const funding = paintGridCell(
      'funding',
      { definite: true, value: 0.0005, text: '+0.05%' },
      1,
      'blank',
      (venues) => venues,
    );
    expect(funding.tone).toBe('up');
    expect(funding.strength).toBe(1);

    const quiet = paintGridCell(
      'liquidations',
      { definite: true, value: 0, text: '0', heat: 1 },
      1,
      'blank',
      (venues) => venues,
    );
    expect(quiet.tone).toBe('flat');

    const hot = paintGridCell(
      'liquidations',
      { definite: true, value: 4_000_000, text: '4.00M', heat: 4 },
      1,
      'blank',
      (venues) => venues,
    );
    expect(hot.tone).toBe('heat');
    expect(hot.strength).toBe(1);
  });

  it('scales CVD against the largest cell on the grid', () => {
    expect(cvdScale([{ cells: [{ definite: true, value: -20 }, { definite: false, value: 100 }] }])).toBe(20);
    const painted = paintGridCell(
      'cvd',
      { definite: true, value: -20, text: '-20' },
      20,
      'blank',
      (venues) => venues,
    );
    expect(painted.tone).toBe('down');
    expect(painted.strength).toBe(1);
  });
});

describe('gridDetailVenues', () => {
  it('keeps each exchange number and does not turn a missing venue into zero', () => {
    expect(
      gridDetailVenues({
        definite: false,
        missing: ['okx'],
        venues: [
          { exchange: 'binance', definite: true, value: 0.0001, text: '+0.0100%' },
          { exchange: 'bybit', definite: true, value: 0, text: '0' },
          { exchange: 'okx', definite: false, value: 5, text: 'leak' },
        ],
      }),
    ).toEqual([
      { exchange: 'binance', text: '+0.0100%', definite: true },
      { exchange: 'bybit', text: '0', definite: true },
      { exchange: 'okx', text: '', definite: false },
    ]);
  });

  it('lists missing venues when the payload has no breakdown', () => {
    expect(gridDetailVenues({ definite: false, missing: ['okx'] })).toEqual([
      { exchange: 'okx', text: '', definite: false },
    ]);
    expect(gridDetailVenues({ definite: true, value: 1, text: '+1' })).toEqual([]);
  });
});

describe('gridColumnLabel', () => {
  it('labels hours and UTC days', () => {
    expect(gridColumnLabel('2026-10-10T15:00:00Z', '24h', 'en-GB')).toBe('15:00');
    expect(gridColumnLabel('2026-10-10T00:00:00Z', '7d', 'en-GB')).toBe('10 Oct');
  });
});
