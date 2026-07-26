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

/** Human-readable "Showing 1–50 of 1,234" for results bar / pagination. */
export function formatResultsRange(offset: number, limit: number, total: number): string {
  if (total <= 0) return '0 matches';
  const from = Math.min(offset + 1, total);
  const to = Math.min(offset + limit, total);
  return `Showing ${from.toLocaleString()}–${to.toLocaleString()} of ${total.toLocaleString()}`;
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
