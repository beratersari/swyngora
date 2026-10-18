import { describe, expect, it } from 'vitest';
import { compactSituationMatch, situationMatchFromSearch } from './helpers';

describe('situation match helpers', () => {
  it('keeps known kinds and drops the rest', () => {
    expect(
      compactSituationMatch({
        priceCvd: 'price_up_cvd_down',
        funding: 'sideways',
        prints: ' ',
      }),
    ).toEqual({ priceCvd: 'price_up_cvd_down' });
    expect(compactSituationMatch({})).toBeUndefined();
  });

  it('reads only known query kinds', () => {
    const params = new URLSearchParams(
      'priceCvd=price_up_cvd_down&funding=sideways&crowd=top_more_long&prints=',
    );
    expect(situationMatchFromSearch(params)).toEqual({
      priceCvd: 'price_up_cvd_down',
      crowd: 'top_more_long',
    });
  });
});
