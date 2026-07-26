import type { SortOrder } from 'antd/es/table/interface';
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
