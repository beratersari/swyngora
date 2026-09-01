export type LiquidationTreemapCoin = {
  symbol?: string;
  base?: string;
  longNotional?: string;
  shortNotional?: string;
  totalNotional?: string;
  count?: number;
};

export type LiquidationTreemapTile = LiquidationTreemapCoin & {
  symbol: string;
  weight: number;
  longShare: number;
  x: number;
  y: number;
  w: number;
  h: number;
};

export type LiquidationTreemapProps = {
  coins: LiquidationTreemapCoin[];
  isLoading?: boolean;
  onOpen?: (symbol: string) => void;
};
