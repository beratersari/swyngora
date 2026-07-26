export type ExchangeChipsProps = {
  exchanges: string[];
  selected: string;
  onSelect: (exchange: string) => void;
  isLoading?: boolean;
};
