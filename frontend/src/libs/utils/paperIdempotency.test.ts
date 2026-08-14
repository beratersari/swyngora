import { describe, expect, it } from 'vitest';
import { newPaperIdempotencyKey } from './paperIdempotency';

describe('newPaperIdempotencyKey', () => {
  it('prefixes and stays within length', () => {
    const k = newPaperIdempotencyKey('web');
    expect(k.startsWith('web-')).toBe(true);
    expect(k.length).toBeLessThanOrEqual(128);
  });

  it('returns distinct keys', () => {
    const a = newPaperIdempotencyKey('detail');
    const b = newPaperIdempotencyKey('detail');
    expect(a).not.toBe(b);
  });
});
