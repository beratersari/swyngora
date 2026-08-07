import type { ReactNode } from 'react';

export type BrandTagVariant = 'status' | 'exchange' | 'paused' | 'live' | 'delist';

export type BrandTagProps = {
  variant?: BrandTagVariant;
  children: ReactNode;
  className?: string;
};
