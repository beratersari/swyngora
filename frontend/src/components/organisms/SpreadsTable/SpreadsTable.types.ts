export type SpreadRow = {
  exchange: string;
  symbol: string;
  current: string;
  typical: string;
  ratio: string;
};

export type SpreadsTableProps = {
  rows: SpreadRow[];
  loading?: boolean;
  hasLoaded: boolean;
  emptyTitle: string;
  emptyHint: string;
  columns: {
    symbol: string;
    exchange: string;
    current: string;
    typical: string;
    ratio: string;
  };
  onRowOpen: (exchange: string, symbol: string) => void;
};
