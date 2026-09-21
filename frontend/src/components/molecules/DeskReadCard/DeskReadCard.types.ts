import type { ReactNode } from 'react';

export type DeskReadCardProps = {
  title: string;
  value: ReactNode;
  caption?: ReactNode;
  visual: ReactNode;
  isLoading?: boolean;
  onClick: () => void;
};
