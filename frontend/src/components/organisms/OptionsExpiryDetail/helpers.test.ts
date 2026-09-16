import { describe, expect, it } from 'vitest';
import { formatMovePct, formatSignedCompact, maxPayout, payoutPct } from './helpers';

describe('options expiry detail helpers', () => {
  it('formats the expected-move percent', () => {
    expect(formatMovePct('13.6')).toBe('±13.6%');
    expect(formatMovePct(undefined)).toBe('—');
  });

  it('formats signed GEX', () => {
    expect(formatSignedCompact('-1250000')).toBe('−1.25M');
    expect(formatSignedCompact(undefined)).toBe('—');
  });

  it('scales payout bars', () => {
    expect(maxPayout([{ payout: '10' }, { payout: '40' }])).toBe(40);
    expect(payoutPct('20', 40)).toBe(50);
  });
});
