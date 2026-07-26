import type { CSSProperties, ReactNode } from 'react';
import type { WithLoadingProps } from '@/components/types';

export type SkeletonVariant =
  'text' | 'title' | 'button' | 'avatar' | 'image' | 'chart' | 'card' | 'input';

export type SkeletonProps = WithLoadingProps & {
  /** Shape preset */
  variant?: SkeletonVariant;
  /** Force active shimmer */
  active?: boolean;
  width?: number | string;
  height?: number | string;
  /** When used as wrapper: if isLoading, show skeleton; else children */
  children?: ReactNode;
  className?: string;
  style?: CSSProperties;
  /** Number of text rows (text variant) */
  rows?: number;
  /** Accessible label while loading */
  'aria-label'?: string;
};
