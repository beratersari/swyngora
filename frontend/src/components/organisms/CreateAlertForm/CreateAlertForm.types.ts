import type { MarketExchange } from '@/libs/api';

export type AlertKind =
  | 'price'
  | 'liquidation_feed'
  | 'liquidation_cascade'
  | 'liquidation_notional'
  | 'liquidation_level'
  | 'setup_radar'
  | 'funding'
  | 'open_interest'
  | 'taker'
  | 'fear_greed';

export type CreateAlertFormProps = {
  defaultExchange?: MarketExchange | string;
  defaultSymbol?: string;
  defaultKind?: AlertKind;
};
