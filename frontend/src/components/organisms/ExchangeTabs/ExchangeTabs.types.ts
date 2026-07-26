import type { MarketExchange } from '@/libs/api';
import type { WithLoadingProps } from '@/components/types';

export type ExchangeTabsProps = WithLoadingProps & {
  exchanges: string[];
  value: MarketExchange;
  onChange: (exchange: MarketExchange) => void;
};
