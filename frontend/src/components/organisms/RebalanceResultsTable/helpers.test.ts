import { describe, expect, it } from 'vitest';
import { formatRebalanceMoney, formatRebalancePct, rebalanceStatusKey } from './helpers';

describe('rebalance result helpers', () => {
  it('formats leftover mix', () => {
    expect(formatRebalancePct(4.2)).toBe('4.20%');
    expect(formatRebalanceMoney(400, 'USDT')).toContain('USDT');
  });

  it('maps status keys', () => {
    expect(rebalanceStatusKey('partial')).toBe('rebalance.statusPartial');
    expect(rebalanceStatusKey('completed')).toBe('rebalance.statusCompleted');
  });
});
