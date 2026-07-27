import { semanticColors } from '@/styles/tokens';
import { ICON_SIZES } from './constants';
import type { IconProps } from './Icon.types';

/**
 * Thin Lucide wrapper — consistent sizes/colors across Atomic UI.
 * Import icons from `lucide-react-native` and pass as `icon={Star}`.
 */
export function Icon({
  icon: LucideIcon,
  size = 'md',
  color = semanticColors.text.primary,
  strokeWidth = 2,
  fill = 'none',
  accessibilityLabel,
  lucideProps,
}: IconProps) {
  const px = typeof size === 'number' ? size : ICON_SIZES[size];
  return (
    <LucideIcon
      size={px}
      color={color}
      strokeWidth={strokeWidth}
      fill={fill}
      accessibilityLabel={accessibilityLabel}
      {...lucideProps}
    />
  );
}
