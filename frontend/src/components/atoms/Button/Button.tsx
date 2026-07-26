import { Skeleton } from '@/components/atoms/Skeleton';
import type { ButtonProps } from './Button.types';
import { StyledButton } from './Button.styles';

/**
 * Atomic Button wrapping Ant Design via styled-components.
 * - `isLoading` → skeleton
 * - `pending` → ant spinner on the button
 */
export function Button({ isLoading = false, pending, disabled, ...rest }: ButtonProps) {
  if (isLoading) {
    return <Skeleton variant="button" active aria-label="Loading button" />;
  }

  return <StyledButton loading={pending} disabled={disabled ?? pending} {...rest} />;
}
