import type { FillPocketMark } from '@/libs/api/endpoints/portfolioApi';

export type { FillPocketMark };

export type FillPocketProps = {
  mark?: FillPocketMark | null;
};

export type FillPocketLineRole = 'fill' | 'pocket' | 'margin';

export type FillPocketPriceLine = {
  id: string;
  price: number;
  role: FillPocketLineRole;
  title: string;
};
