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
