import { describe, expect, it } from 'vitest';
import {
  formatResultsRange,
  paginationChanged,
  resolveTableChangeAction,
} from './MarketsTable.helpers';

describe('formatResultsRange', () => {
  it('formats empty and first page', () => {
    expect(formatResultsRange(0, 50, 0)).toBe('0 matches');
    expect(formatResultsRange(0, 50, 120)).toBe('Showing 1–50 of 120');
  });

  it('formats later pages and clamps end', () => {
    expect(formatResultsRange(50, 50, 120)).toBe('Showing 51–100 of 120');
    expect(formatResultsRange(100, 50, 120)).toBe('Showing 101–120 of 120');
  });
});

describe('resolveTableChangeAction', () => {
  it('trusts antd extra.action for paginate even if sort looks active', () => {
    expect(resolveTableChangeAction('paginate', true, true)).toBe('paginate');
    expect(resolveTableChangeAction('sort', false, true)).toBe('sort');
  });

  it('falls back when action missing', () => {
    expect(resolveTableChangeAction(undefined, true, true)).toBe('sort');
    expect(resolveTableChangeAction(undefined, false, true)).toBe('paginate');
    expect(resolveTableChangeAction(undefined, false, false)).toBe('none');
  });
});

describe('paginationChanged', () => {
  it('detects page and pageSize changes', () => {
    expect(paginationChanged({ current: 2, pageSize: 50 }, 0, 50)).toEqual({
      changed: true,
      nextOffset: 50,
      nextLimit: 50,
    });
    expect(paginationChanged({ current: 1, pageSize: 50 }, 0, 50)).toEqual({
      changed: false,
      nextOffset: 0,
      nextLimit: 50,
    });
  });
});
