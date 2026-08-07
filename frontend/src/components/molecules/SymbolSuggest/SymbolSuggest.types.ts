import type { CSSProperties } from 'react';
import type { MarketExchange } from '@/libs/api';

export type SymbolSuggestProps = {
  exchange: MarketExchange | string;
  value: string;
  onChange: (symbol: string) => void;
  placeholder?: string;
  disabled?: boolean;
  'aria-label'?: string;
  style?: CSSProperties;
};
