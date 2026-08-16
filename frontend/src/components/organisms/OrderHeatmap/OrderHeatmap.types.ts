export type OrderHeatmapLevel = {
  price?: string;
  notional?: string;
  isWall?: boolean;
};

export type OrderHeatmapColumn = {
  t?: string;
  mid?: string;
  bids?: OrderHeatmapLevel[];
  asks?: OrderHeatmapLevel[];
};

export type OrderHeatmapData = {
  exchange?: string;
  symbol?: string;
  groupSize?: string;
  windowSeconds?: number;
  sampleEveryMs?: number;
  from?: string;
  to?: string;
  columns?: OrderHeatmapColumn[];
  live?: boolean;
};

export type OrderHeatmapProps = {
  data?: OrderHeatmapData;
  windowSeconds: number;
  onWindowChange: (seconds: number) => void;
  isLoading?: boolean;
  isFetching?: boolean;
  errorMessage?: string | null;
};

export type HeatSide = 'bid' | 'ask';

export type HeatCell = {
  timeMs: number;
  price: number;
  notional: number;
  isWall: boolean;
  side: HeatSide;
};

export type HeatHover = {
  x: number;
  y: number;
  timeMs: number;
  price: number;
  mid: number | null;
  bid: number;
  ask: number;
  bidWall: boolean;
  askWall: boolean;
};

export type ColumnRect = {
  index: number;
  x: number;
  w: number;
  timeMs: number;
  isCob: boolean;
};

export type HeatLayout = {
  plotX: number;
  plotY: number;
  plotW: number;
  plotH: number;
  scaleX: number;
  scaleW: number;
  minPrice: number;
  maxPrice: number;
  step: number;
  prices: number[];
  rects: ColumnRect[];
  peak: number;
};
