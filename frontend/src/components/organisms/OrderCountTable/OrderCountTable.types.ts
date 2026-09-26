export type OrderCountRow = {
  exchange: string;
  symbol: string;
  ratio: string;
  lean: string;
  buyCount: string;
  sellCount: string;
  summary: string;
};

export type OrderCountTableProps = {
  rows: OrderCountRow[];
  loading?: boolean;
  hasLoaded: boolean;
  emptyHint: string;
  emptyTitle: string;
  columns: {
    symbol: string;
    ratio: string;
    lean: string;
    buyCount: string;
    sellCount: string;
    summary: string;
  };
  onRowOpen: (exchange: string, symbol: string) => void;
};
