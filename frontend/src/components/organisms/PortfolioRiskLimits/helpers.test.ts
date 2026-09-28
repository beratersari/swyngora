import { describe, expect, it } from 'vitest';
import { exposureLine } from './helpers';

describe('exposureLine', () => {
  it('shows the live share and the cap', () => {
    expect(exposureLine(72.5, 60)).toBe('72.5% · 60%');
  });

  it('omits the cap when the rule is off', () => {
    expect(exposureLine(12, null)).toBe('12%');
    expect(exposureLine(undefined, 60)).toBe('');
  });
});
