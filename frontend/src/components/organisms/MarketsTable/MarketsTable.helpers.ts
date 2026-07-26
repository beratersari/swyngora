import type { SortOrder, TableCurrentDataSource } from 'antd/es/table/interface';
import type { TablePaginationConfig } from 'antd/es/table';
import type { SpotSortOrder } from '@/libs/api';

export function toAntdSortOrder(order: SpotSortOrder, active: boolean): SortOrder {
  if (!active) return null;
  return order === 'asc' ? 'ascend' : 'descend';
}

export function fromAntdSortOrder(order: SortOrder): SpotSortOrder | null {
  if (order === 'ascend') return 'asc';
  if (order === 'descend') return 'desc';
  return null;
}

/** Toggle API sort order (asc ↔ desc). */
export function toggleSortOrder(order: SpotSortOrder): SpotSortOrder {
  return order === 'asc' ? 'desc' : 'asc';
}

/**
 * Resolve the next API sort order from an Ant Design sorter event.
 *
 * Ant Design may emit `order: undefined` and wipe `columnKey`/`field` when the
 * internal next direction is out of range — callers must then fall back to
 * toggling the active column (see resolveSortChange).
 */
export function resolveNextSortOrder(
  antdOrder: SortOrder | undefined | null,
  currentOrder: SpotSortOrder,
  opts?: { isActiveColumn?: boolean; actionIsSort?: boolean },
): SpotSortOrder | null {
  const mapped = fromAntdSortOrder(antdOrder ?? null);
  if (mapped) return mapped;
  if (opts?.isActiveColumn) {
    return toggleSortOrder(currentOrder);
  }
  if (opts?.actionIsSort) {
    // Unknown column with cleared order: keep cycling the active sort.
    return toggleSortOrder(currentOrder);
  }
  return null;
}

export type SortChangeResult =
  | { type: 'sort'; field: string; order: SpotSortOrder }
  | { type: 'none' };

/**
 * Fully resolve a table sort click into an API sort field + order.
 * Handles antd's empty sorter payload after the last sortDirections entry.
 */
export function resolveSortChange(input: {
  columnKey?: string | number | null;
  field?: string | number | readonly (string | number)[] | null;
  antdOrder?: SortOrder | null;
  action?: string;
  activeSort: string;
  activeOrder: SpotSortOrder;
  columnSortMap: Record<string, string>;
}): SortChangeResult {
  const columnId =
    input.columnKey != null && input.columnKey !== ''
      ? String(input.columnKey)
      : input.field != null && !Array.isArray(input.field)
        ? String(input.field)
        : undefined;

  const mappedField = columnId ? input.columnSortMap[columnId] : undefined;
  const isSortAction = input.action === 'sort';

  // Ant Design wiped identifiers after descend with a short sortDirections list.
  if (!mappedField) {
    if (isSortAction) {
      return {
        type: 'sort',
        field: input.activeSort,
        order: toggleSortOrder(input.activeOrder),
      };
    }
    return { type: 'none' };
  }

  const nextOrder = resolveNextSortOrder(input.antdOrder, input.activeOrder, {
    isActiveColumn: mappedField === input.activeSort,
    actionIsSort: isSortAction,
  });

  if (!nextOrder) return { type: 'none' };

  // No-op only when antd echoes the same sort we already have.
  if (mappedField === input.activeSort && nextOrder === input.activeOrder) {
    if (isSortAction) {
      return {
        type: 'sort',
        field: mappedField,
        order: toggleSortOrder(input.activeOrder),
      };
    }
    return { type: 'none' };
  }

  return { type: 'sort', field: mappedField, order: nextOrder };
}

/** Structured range for i18n (UI formats via t('markets:results.range')). */
export type ResultsRange =
  | { kind: 'empty' }
  | { kind: 'range'; from: number; to: number; total: number };

export function getResultsRange(offset: number, limit: number, total: number): ResultsRange {
  if (total <= 0) return { kind: 'empty' };
  const from = Math.min(offset + 1, total);
  const to = Math.min(offset + limit, total);
  return { kind: 'range', from, to, total };
}

/** @deprecated Prefer getResultsRange + t() for localization */
export function formatResultsRange(offset: number, limit: number, total: number): string {
  const r = getResultsRange(offset, limit, total);
  if (r.kind === 'empty') return '0 matches';
  return `Showing ${r.from.toLocaleString()}–${r.to.toLocaleString()} of ${r.total.toLocaleString()}`;
}

export type TableChangeAction = NonNullable<TableCurrentDataSource<unknown>['action']>;

/**
 * Decide whether Ant Design Table onChange is pagination vs sort.
 * When sort is controlled, sorter is always populated — so we must prefer
 * `extra.action` and never treat every change as sort (that blocked paging).
 */
export function resolveTableChangeAction(
  extraAction: TableChangeAction | undefined,
  sortChanged: boolean,
  pageChanged: boolean,
): 'sort' | 'paginate' | 'filter' | 'none' {
  if (extraAction === 'sort' || extraAction === 'paginate' || extraAction === 'filter') {
    return extraAction;
  }
  // Fallback when action missing (older antd / tests)
  if (sortChanged) return 'sort';
  if (pageChanged) return 'paginate';
  return 'none';
}

export function paginationChanged(
  pag: TablePaginationConfig,
  offset: number,
  limit: number,
): { changed: boolean; nextOffset: number; nextLimit: number } {
  const nextLimit = pag.pageSize ?? limit;
  const page = pag.current ?? 1;
  const nextOffset = (page - 1) * nextLimit;
  return {
    changed: nextOffset !== offset || nextLimit !== limit,
    nextOffset,
    nextLimit,
  };
}
