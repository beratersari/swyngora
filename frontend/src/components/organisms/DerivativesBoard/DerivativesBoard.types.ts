import type { DerivativesCoin } from '@/libs/api';

export type DerivativesBoardProps = {
  exchange: string;
  coins: DerivativesCoin[];
  loading?: boolean;
};

export const BOARD_SORT_KEYS = [
  'base',
  'price',
  'funding',
  'openInterest',
  'change1h',
  'change24h',
  'liqLong',
  'liqShort',
  'longPct',
  'takerBuyPct',
  'gapPct',
  'cvdDay',
  'cvdSplit',
  'printChange',
  'orderCount',
  'unusual',
  'liqHeat',
  'bidPct',
] as const;

export type BoardSortKey = (typeof BOARD_SORT_KEYS)[number];

export type BoardSortDir = 'asc' | 'desc';

export type BoardSort = {
  key: BoardSortKey;
  dir: BoardSortDir;
};
