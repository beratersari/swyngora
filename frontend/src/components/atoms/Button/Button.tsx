import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import type { ButtonProps } from './Button.types';
import { StyledButton } from './Button.styles';

/**
 * Atomic Button wrapping Ant Design via styled-components.
 * - `isLoading` → skeleton
 * - `pending` → ant spinner; button is disabled unless `disabled` is set
 */
export function Button({ isLoading = false, pending, disabled, ...rest }: ButtonProps) {
  const { t } = useTranslation('common');
  if (isLoading) {
    return <Skeleton variant="button" active aria-label={t('a11y.loadingButton')} />;
  }

  // pending always disables unless caller forces `disabled` (explicit true/false wins).
  const isDisabled = disabled !== undefined ? disabled : Boolean(pending);
  return <StyledButton loading={pending} disabled={isDisabled} {...rest} />;
}
