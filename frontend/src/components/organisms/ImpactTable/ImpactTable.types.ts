export type ImpactRow = {
  exchange: string;
  symbol: string;
  slippage: string;
  versusBest: string;
  impact: string;
  exhausted: string;
};

export type ImpactTableProps = {
  rows: ImpactRow[];
  loading?: boolean;
  hasLoaded: boolean;
  emptyHint: string;
  emptyTitle: string;
  columns: {
    symbol: string;
    slippage: string;
    versusBest: string;
    impact: string;
    exhausted: string;
  };
  onRowOpen: (exchange: string, symbol: string) => void;
};
