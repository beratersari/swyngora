import type { ElementType } from 'react';
import { Skeleton } from '@/components/atoms/Skeleton';
import { DEFAULT_COLOR, DEFAULT_VARIANT } from './Text.constants';
import { defaultTagForVariant, variantStyle } from './Text.helpers';
import type { TextProps } from './Text.types';
import { TextRoot } from './Text.styles';

/**
 * Design-system text atom — type scale + palette via styled-components.
 * Supports `isLoading` → skeleton line.
 */
export function Text({
  variant = DEFAULT_VARIANT,
  color = DEFAULT_COLOR,
  as,
  weight,
  truncate,
  mono,
  isLoading = false,
  skeletonWidth = '80%',
  children,
  className,
  style,
  ...rest
}: TextProps) {
  if (isLoading) {
    return (
      <Skeleton
        variant={variant === 'display' || variant.startsWith('h') ? 'title' : 'text'}
        width={skeletonWidth}
        active
      />
    );
  }

  const tag = (as ?? defaultTagForVariant(variant)) as ElementType;
  const computed = variantStyle(variant, { weight, mono, color });

  return (
    <TextRoot
      as={tag}
      $truncate={truncate}
      $mono={mono}
      className={className}
      style={{ ...computed, ...style }}
      {...rest}
    >
      {children}
    </TextRoot>
  );
}
