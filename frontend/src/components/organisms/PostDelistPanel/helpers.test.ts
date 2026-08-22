import { describe, expect, it } from 'vitest';
import { formatChangePct } from './helpers';

describe('formatChangePct', () => {
  it('formats signed percent', () => {
    expect(formatChangePct('4.25')).toBe('+4.25%');
    expect(formatChangePct('-3.1')).toBe('-3.10%');
  });

  it('falls back for empty', () => {
    expect(formatChangePct(null)).toBe('—');
    expect(formatChangePct('x')).toBe('—');
  });
});
