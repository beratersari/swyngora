import type { SpotPosition } from '@/libs/api';

export type PortfolioPositionsTableProps = {
  items: SpotPosition[];
  loading?: boolean;
  portfolioId?: string;
  onOpen?: (exchange: string, symbol: string) => void;
};
