export type PumpEventRowViewModel = {
  id: string;
  returnLabel: string;
  returnTone: 'success' | 'error' | 'secondary';
  timeLabel: string;
  metaLabel: string;
};

export type PumpEventListProps = {
  title?: string;
  subtitle?: string | null;
  rows: PumpEventRowViewModel[];
  isLoading?: boolean;
  errorMessage?: string | null;
  emptyMessage?: string | null;
  disclaimer?: string | null;
};
