export type MarketsPaginationProps = {
  offset: number;
  limit: number;
  total: number;
  canPrev: boolean;
  canNext: boolean;
  onPrev: () => void;
  onNext: () => void;
};
