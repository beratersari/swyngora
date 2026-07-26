import type { CSSProperties, ElementType, HTMLAttributes, ReactNode } from 'react';
import type { TextColor, TypeVariant } from '@/styles/tokens';
import type { WithLoadingProps } from '@/components/types';

export type TextProps = Omit<HTMLAttributes<HTMLElement>, 'color'> &
  WithLoadingProps & {
    /** Typography role from the design system type scale */
    variant?: TypeVariant;
    /** Semantic / brand text color */
    color?: TextColor;
    /** Render as another element (default inferred from variant) */
    as?: ElementType;
    /** Bold override */
    weight?: 400 | 500 | 600 | 700;
    /** Truncate with ellipsis */
    truncate?: boolean;
    /** Mono / tabular for prices when not using numeric variant */
    mono?: boolean;
    children?: ReactNode;
    className?: string;
    style?: CSSProperties;
    /** Skeleton width when isLoading (default 80%) */
    skeletonWidth?: number | string;
  };
