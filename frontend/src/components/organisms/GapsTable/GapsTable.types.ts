export type GapRow = {
  exchange: string;
  symbol: string;
  easier: string;
  above: string;
  below: string;
  emptyAbove: string;
  emptyBelow: string;
  widest: string;
};

export type GapsTableProps = {
  rows: GapRow[];
  loading?: boolean;
  hasLoaded: boolean;
  emptyHint: string;
  emptyTitle: string;
  columns: {
    symbol: string;
    easier: string;
    above: string;
    below: string;
    emptyAbove: string;
    emptyBelow: string;
    widest: string;
  };
  onRowOpen: (exchange: string, symbol: string) => void;
};
