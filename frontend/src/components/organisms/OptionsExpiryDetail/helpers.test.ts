import { describe, expect, it } from 'vitest';
import { formatMovePct, maxPayout, payoutPct } from './helpers';

describe('options expiry detail helpers', () => {
  it('formats the expected-move percent', () => {
    expect(formatMovePct('13.6')).toBe('±13.6%');
    expect(formatMovePct(undefined)).toBe('—');
  });

  it('scales payout bars', () => {
    expect(maxPayout([{ payout: '10' }, { payout: '40' }])).toBe(40);
    expect(payoutPct('20', 40)).toBe(50);
  });
});
