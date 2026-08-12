import type { PaperTrade } from '@/libs/api';

export type PortfolioTradesTableProps = {
  items: PaperTrade[];
  loading?: boolean;
  onOpen?: (exchange: string, symbol: string) => void;
};
