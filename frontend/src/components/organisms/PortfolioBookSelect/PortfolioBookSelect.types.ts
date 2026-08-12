import type { PortfolioSummary } from '@/libs/api';

export type PortfolioBookSelectProps = {
  books: PortfolioSummary[];
  selectedId?: string;
  loading?: boolean;
  creating?: boolean;
  onSelect: (id: string) => void;
  onCreate: (input: { name: string; startingBalance: number }) => Promise<void>;
};
