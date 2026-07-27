import { describe, expect, it } from 'vitest';
import { compactParams } from './marketApi';

describe('compactParams', () => {
  it('drops undefined null and empty string', () => {
    expect(
      compactParams({
        a: 'x',
        b: '',
        c: undefined,
        d: null,
        e: 0,
        f: 12,
      }),
    ).toEqual({ a: 'x', e: 0, f: 12 });
  });

  it('ignores non string/number values', () => {
    expect(compactParams({ ok: 'yes', skip: true as unknown as string })).toEqual({
      ok: 'yes',
    });
  });
});
