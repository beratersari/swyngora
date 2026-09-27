export type WallRow = {
  exchange: string;
  symbol: string;
  side: string;
  price: string;
  notional: string;
  distance: string;
  kind: string;
  appears: string;
  present: string;
};

export type WallsTableProps = {
  rows: WallRow[];
  loading?: boolean;
  hasLoaded: boolean;
  emptyHint: string;
  emptyTitle: string;
  columns: {
    symbol: string;
    venue: string;
    side: string;
    price: string;
    notional: string;
    distance: string;
    kind: string;
    appears: string;
    present: string;
  };
  onRowOpen: (exchange: string, symbol: string) => void;
};
