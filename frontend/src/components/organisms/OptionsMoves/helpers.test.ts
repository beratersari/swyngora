import { describe, expect, it } from 'vitest';
import { unusualMoves } from './helpers';

describe('unusualMoves', () => {
  it('keeps titled findings only', () => {
    expect(
      unusualMoves([
        { title: 'Open interest jumped 12% in 1h — well above typical', kind: 'oi_jump' },
        { title: '' },
      ]),
    ).toHaveLength(1);
  });

  it('is empty when nothing stands out', () => {
    expect(unusualMoves(undefined)).toEqual([]);
    expect(unusualMoves([])).toEqual([]);
  });
});
