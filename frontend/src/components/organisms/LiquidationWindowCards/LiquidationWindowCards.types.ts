export type LiquidationCardWindow = {
  window?: string;
  longNotional?: string;
  shortNotional?: string;
  totalNotional?: string;
  count?: number;
  complete?: boolean;
};

export type LiquidationWindowCardsProps = {
  windows: LiquidationCardWindow[];
  selectedWindow: string;
  onSelect: (window: string) => void;
  isLoading?: boolean;
};
