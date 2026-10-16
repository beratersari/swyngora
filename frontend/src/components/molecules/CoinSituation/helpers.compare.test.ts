import { describe, expect, it } from 'vitest';
import { compareCoinSituation } from './helpers';

describe('compareCoinSituation', () => {
  it('ignores a clause the entry did not store', () => {
    const got = compareCoinSituation(
      { priceCvd: { kind: 'price_up_cvd_down' } },
      { priceCvd: { kind: 'price_up_cvd_down' }, funding: { payer: 'long' } },
    );
    expect(got.same).toBe(true);
    expect(got.clauses).toEqual([]);
  });

  it('names a stored clause that changed', () => {
    const got = compareCoinSituation(
      { priceCvd: { kind: 'price_up_cvd_down' }, openInterest: { flow: 'opened' } },
      { priceCvd: { kind: 'agrees' }, openInterest: { flow: 'opened' } },
    );
    expect(got.same).toBe(false);
    expect(got.clauses).toEqual([
      {
        clause: 'priceCvd',
        status: 'changed',
        entry: 'price_up_cvd_down',
        live: 'agrees',
      },
    ]);
  });
});
