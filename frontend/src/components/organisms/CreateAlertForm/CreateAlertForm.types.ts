import type { MarketExchange } from '@/libs/api';

export type CreateAlertFormProps = {
  defaultExchange?: MarketExchange | string;
  defaultSymbol?: string;
};
