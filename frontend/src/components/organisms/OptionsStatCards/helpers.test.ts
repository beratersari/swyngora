import { describe, expect, it } from 'vitest';
import { formatIv, formatPcr } from './helpers';

describe('options stat helpers', () => {
  it('formats implied vol as a percent', () => {
    expect(formatIv('49.52')).toBe('49.5%');
    expect(formatIv(undefined)).toBe('—');
  });

  it('formats put/call to two decimals', () => {
    expect(formatPcr('0.4')).toBe('0.40');
    expect(formatPcr('')).toBe('—');
  });
});
