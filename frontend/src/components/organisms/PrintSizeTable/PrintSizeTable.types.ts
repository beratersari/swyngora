export type PrintSizeRow = {
  exchange: string;
  symbol: string;
  buyAvg: string;
  buyPrior: string;
  sellAvg: string;
  sellPrior: string;
  read: string;
};

export type PrintSizeTableProps = {
  rows: PrintSizeRow[];
  loading?: boolean;
  hasLoaded: boolean;
  emptyHint: string;
  emptyTitle: string;
  columns: {
    symbol: string;
    buyAvg: string;
    buyPrior: string;
    sellAvg: string;
    sellPrior: string;
    read: string;
  };
  onRowOpen: (exchange: string, symbol: string) => void;
};
