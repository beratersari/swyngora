import { describe, expect, it } from 'vitest';
import {
  formatResultsRange,
  getResultsRange,
  paginationChanged,
  resolveNextSortOrder,
  resolveSortChange,
  resolveTableChangeAction,
  toggleSortOrder,
} from './MarketsTable.helpers';
import { COLUMN_SORT } from './MarketsTable.constants';

describe('getResultsRange', () => {
  it('returns empty and range parts', () => {
    expect(getResultsRange(0, 50, 0)).toEqual({ kind: 'empty' });
    expect(getResultsRange(0, 50, 120)).toEqual({ kind: 'range', from: 1, to: 50, total: 120 });
    expect(getResultsRange(100, 50, 120)).toEqual({ kind: 'range', from: 101, to: 120, total: 120 });
  });
});

describe('formatResultsRange (legacy English)', () => {
  it('formats empty and first page', () => {
    expect(formatResultsRange(0, 50, 0)).toBe('0 matches');
    expect(formatResultsRange(0, 50, 120)).toBe('Showing 1–50 of 120');
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

describe('resolveNextSortOrder', () => {
  it('maps ascend/descend to api orders', () => {
    expect(resolveNextSortOrder('ascend', 'desc')).toBe('asc');
    expect(resolveNextSortOrder('descend', 'asc')).toBe('desc');
  });

  it('toggles when antd clears the active sort column', () => {
    expect(
      resolveNextSortOrder(null, 'desc', { isActiveColumn: true }),
    ).toBe('asc');
    expect(
      resolveNextSortOrder(null, 'asc', { isActiveColumn: true }),
    ).toBe('desc');
  });

  it('toggles on sort action when order is cleared', () => {
    expect(
      resolveNextSortOrder(null, 'asc', { actionIsSort: true }),
    ).toBe('desc');
  });

  it('returns null when cleared without sort context', () => {
    expect(resolveNextSortOrder(null, 'desc')).toBeNull();
  });
});

describe('toggleSortOrder', () => {
  it('flips asc and desc', () => {
    expect(toggleSortOrder('asc')).toBe('desc');
    expect(toggleSortOrder('desc')).toBe('asc');
  });
});

describe('resolveSortChange', () => {
  const base = {
    activeSort: 'quoteVolume',
    activeOrder: 'desc' as const,
    columnSortMap: COLUMN_SORT as Record<string, string>,
  };

  it('maps a normal ascend click on a column', () => {
    expect(
      resolveSortChange({
        ...base,
        columnKey: 'lastPrice',
        antdOrder: 'ascend',
        action: 'sort',
      }),
    ).toEqual({ type: 'sort', field: 'lastPrice', order: 'asc' });
  });

  it('cycles descend → asc when antd wipes columnKey (third-click bug)', () => {
    expect(
      resolveSortChange({
        ...base,
        activeOrder: 'desc',
        columnKey: undefined,
        field: undefined,
        antdOrder: null,
        action: 'sort',
      }),
    ).toEqual({ type: 'sort', field: 'quoteVolume', order: 'asc' });
  });

  it('toggles when antd echoes the same order again', () => {
    expect(
      resolveSortChange({
        ...base,
        activeSort: 'lastPrice',
        activeOrder: 'desc',
        columnKey: 'lastPrice',
        antdOrder: 'descend',
        action: 'sort',
      }),
    ).toEqual({ type: 'sort', field: 'lastPrice', order: 'asc' });
  });

  it('returns none when not a sort action and identifiers missing', () => {
    expect(
      resolveSortChange({
        ...base,
        action: 'paginate',
        columnKey: undefined,
        antdOrder: null,
      }),
    ).toEqual({ type: 'none' });
  });
});
