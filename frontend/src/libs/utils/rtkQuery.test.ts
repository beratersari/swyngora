import { describe, expect, it } from 'vitest';
import { rtkCurrent, rtkCurrentPending } from './rtkQuery';

describe('rtkCurrent', () => {
  it('returns currentData only, never the previous-arg .data', () => {
    expect(rtkCurrent({ data: { id: 'a' }, currentData: undefined })).toBeUndefined();
    expect(rtkCurrent({ data: { id: 'a' }, currentData: { id: 'b' } })?.id).toBe('b');
  });

  it('treats arg-change fetch without currentData as pending', () => {
    expect(
      rtkCurrentPending({ data: { id: 'a' }, currentData: undefined, isFetching: true, isLoading: false }),
    ).toBe(true);
    expect(
      rtkCurrentPending({ data: { id: 'a' }, currentData: { id: 'a' }, isFetching: true, isLoading: false }),
    ).toBe(false);
  });
});
