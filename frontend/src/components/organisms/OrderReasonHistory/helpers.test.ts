import { describe, expect, it } from 'vitest';
import { reasonHistoryHasCloses, selectedReasonHistory, type ReasonHistoryRow } from './helpers';

const rows: ReasonHistoryRow[] = [
  { clause: 'priceCvd', kind: 'price_up_cvd_down', still: { count: 1 }, left: { count: 1 } },
  { clause: 'funding', kind: 'long', still: { count: 0 }, left: { count: 0 } },
];

describe('selectedReasonHistory', () => {
  it('keeps only the clauses that are still checked, in server order', () => {
    expect(selectedReasonHistory(rows, ['funding', 'priceCvd']).map((row) => row.clause)).toEqual([
      'priceCvd',
      'funding',
    ]);
    expect(selectedReasonHistory(rows, ['funding']).map((row) => row.clause)).toEqual(['funding']);
    expect(selectedReasonHistory(rows, [])).toEqual([]);
    expect(selectedReasonHistory(undefined, ['priceCvd'])).toEqual([]);
  });

  it('treats two empty groups as no past close', () => {
    expect(reasonHistoryHasCloses(rows[0])).toBe(true);
    expect(reasonHistoryHasCloses(rows[1])).toBe(false);
  });
});
