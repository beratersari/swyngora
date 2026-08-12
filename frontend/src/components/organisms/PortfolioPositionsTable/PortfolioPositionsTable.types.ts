import type { SpotPosition } from '@/libs/api';

export type PortfolioPositionsTableProps = {
  items: SpotPosition[];
  loading?: boolean;
  onOpen?: (exchange: string, symbol: string) => void;
};
