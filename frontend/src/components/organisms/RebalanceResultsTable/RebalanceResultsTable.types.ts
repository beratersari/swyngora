import type { RebalanceResult } from '@/libs/api';

export type RebalanceResultsTableProps = {
  results: RebalanceResult[];
  isLoading?: boolean;
  expandedId?: string;
  onExpand: (id: string | undefined) => void;
  detail?: RebalanceResult;
  detailLoading?: boolean;
};
