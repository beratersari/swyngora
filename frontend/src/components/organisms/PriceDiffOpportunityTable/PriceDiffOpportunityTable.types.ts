import type { PriceDiffOpportunity } from '@/libs/api';

export type PriceDiffOppStatusFilter = 'open' | 'closed' | 'all';

export type PriceDiffOpportunityTableProps = {
  opportunities: PriceDiffOpportunity[];
  isLoading?: boolean;
  status: PriceDiffOppStatusFilter;
  onStatusChange: (status: PriceDiffOppStatusFilter) => void;
  expandedId?: string;
  onExpand: (id: string | undefined) => void;
  detail?: PriceDiffOpportunity;
  detailLoading?: boolean;
};
