import type { DerivativesCoin } from '@/libs/api';

export type DerivativesBoardProps = {
  exchange: string;
  coins: DerivativesCoin[];
  loading?: boolean;
};
