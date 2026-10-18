import { describe, expect, it } from 'vitest';
import { signedQuote } from './helpers';

describe('signedQuote', () => {
  it('marks a higher cost with a plus and keeps a lower cost negative', () => {
    expect(signedQuote(0.4).startsWith('+')).toBe(true);
    expect(signedQuote(-3).startsWith('-') || signedQuote(-3).includes('-')).toBe(true);
    expect(signedQuote(0)).toBe('0');
  });
});
