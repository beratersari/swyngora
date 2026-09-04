import type { MarketExchange } from '@/libs/api';

export type AlertKind =
  | 'price'
  | 'liquidation_feed'
  | 'liquidation_cascade'
  | 'liquidation_notional'
  | 'setup_radar';

export type CreateAlertFormProps = {
  defaultExchange?: MarketExchange | string;
  defaultSymbol?: string;
  defaultKind?: AlertKind;
};
