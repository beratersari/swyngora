export type SpreadOrderRow = {
  id: string;
  symbol: string;
  side: string;
  quantity: number;
  filledQuantity: number;
  sliceCount: number;
  slicesDone: number;
};

export type SpreadOrderListProps = {
  items: SpreadOrderRow[];
  canceling?: boolean;
  onCancel: (id: string) => void;
};
