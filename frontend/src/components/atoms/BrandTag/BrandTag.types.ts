import type { ReactNode } from 'react';

export type BrandTagVariant =
  | 'status'
  | 'exchange'
  | 'paused'
  | 'live'
  | 'delist'
  | 'up'
  | 'down'
  | 'outline'
  | 'gradeA'
  | 'gradeB'
  | 'gradeC';

export type BrandTagProps = {
  variant?: BrandTagVariant;
  children: ReactNode;
  className?: string;
};
