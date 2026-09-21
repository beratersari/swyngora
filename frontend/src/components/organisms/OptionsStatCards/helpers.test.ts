import { describe, expect, it } from 'vitest';
import { formatIv, formatPcr, formatVolGap, optionsStatValues } from './helpers';

describe('options stat helpers', () => {
  it('formats implied vol as a percent', () => {
    expect(formatIv('49.52')).toBe('49.5%');
    expect(formatIv(undefined)).toBe('—');
  });

  it('formats put/call to two decimals', () => {
    expect(formatPcr('0.4')).toBe('0.40');
    expect(formatPcr('')).toBe('—');
  });

  it('formats the DVOL minus realized gap in vol points', () => {
    expect(formatVolGap('25')).toBe('+25.0');
    expect(formatVolGap('-8.2')).toBe('-8.2');
    expect(formatVolGap(undefined)).toBe('');
  });

  it('reads DVOL, realized, and gap from volCompare', () => {
    const vals = optionsStatValues(
      {
        volCompare: { dvol: '70', realized: '45', gap: '+25', days: 30, lean: 'implied_richer' },
        dvol: '70',
      },
      'BTC',
    );
    expect(vals.dvol).toBe('70.0%');
    expect(vals.realized).toBe('45.0%');
    expect(vals.volGap).toBe('+25.0');
  });
});
