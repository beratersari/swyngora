import { describe, expect, it } from 'vitest';
import { vsMarketFindings } from './helpers';

describe('vsMarketFindings', () => {
  it('drops empty titles', () => {
    expect(
      vsMarketFindings([
        { title: 'Options lean put while futures accounts are long', kind: 'puts_vs_futures_long' },
        { title: '' },
      ]),
    ).toHaveLength(1);
  });

  it('returns empty when there is nothing to show', () => {
    expect(vsMarketFindings(undefined)).toEqual([]);
    expect(vsMarketFindings([])).toEqual([]);
  });
});
