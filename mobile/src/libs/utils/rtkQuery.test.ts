import { describe, expect, it } from 'vitest';
import { rtkCurrent, rtkCurrentPending } from './rtkQuery';

describe('rtkCurrent', () => {
  it('uses currentData so a previous arg is not live', () => {
    const q = {
      data: { symbol: 'BTCUSDT' },
      currentData: undefined,
      isFetching: true,
      isLoading: false,
    };
    expect(rtkCurrent(q)).toBeUndefined();
    expect(rtkCurrentPending(q)).toBe(true);
    expect(rtkCurrent({ ...q, currentData: { symbol: 'ETHUSDT' } })?.symbol).toBe('ETHUSDT');
  });
});
