import type { ReactNode } from 'react';

/**
 * Shared loading contract for design-system components.
 * When `isLoading` is true, the component should render a Skeleton
 * (or an equivalent loading affordance) instead of primary content.
 */
export type WithLoadingProps = {
  /** When true, show skeleton / loading UI instead of content */
  isLoading?: boolean;
};

export type WithLoadingChildren = WithLoadingProps & {
  children?: ReactNode;
};
