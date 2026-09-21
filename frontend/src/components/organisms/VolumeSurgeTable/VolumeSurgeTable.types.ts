export type VolumeSurgeRow = {
  exchange: string;
  symbol: string;
  maxRatio: string;
  hottest: string;
  grade: string;
  dominant: string;
  summary: string;
};

export type VolumeSurgeTableProps = {
  rows: VolumeSurgeRow[];
  loading?: boolean;
  hasScanned: boolean;
  emptyHint: string;
  emptyTitle: string;
  columns: {
    symbol: string;
    maxRatio: string;
    hottest: string;
    grade: string;
    dominant: string;
    summary: string;
  };
  onRowOpen: (exchange: string, symbol: string) => void;
};
