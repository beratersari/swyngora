import { describe, expect, it } from 'vitest';
import { formatFundingPct, formatTapeNum, windowByName } from './TapePanel.helpers';

describe('TapePanel helpers', () => {
  it('compacts large notionals', () => {
    expect(formatTapeNum(1_500_000_000)).toBe('1.50B');
    expect(formatTapeNum('')).toBe('—');
  });

  it('formats funding percents', () => {
    expect(formatFundingPct(0.0123)).toBe('0.0123%');
    expect(formatFundingPct(null)).toBe('—');
  });

  it('picks a named window', () => {
    expect(windowByName([{ window: '1h' }, { window: '24h' }], '24h')?.window).toBe('24h');
  });
});
