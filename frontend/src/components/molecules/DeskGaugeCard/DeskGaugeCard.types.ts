import type { ReactNode } from 'react';

export type DeskGaugeCardProps = {
  title: string;
  value: ReactNode;
  caption?: ReactNode;
  gaugeValue?: number | null;
  minLabel: string;
  maxLabel: string;
  isLoading?: boolean;
  onClick: () => void;
};
