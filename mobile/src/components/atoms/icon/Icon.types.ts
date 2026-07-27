import type { LucideIcon, LucideProps } from 'lucide-react-native';
import type { IconSizeName } from './constants';

export type IconProps = {
  /** Lucide icon component, e.g. `Star` from `lucide-react-native`. */
  icon: LucideIcon;
  /** Named size or raw pixel size. Default `md` (20). */
  size?: IconSizeName | number;
  /** Stroke/fill color. Defaults to primary text. */
  color?: string;
  strokeWidth?: number;
  /** SVG fill (e.g. solid star when favorited). */
  fill?: string;
  accessibilityLabel?: string;
  /** Extra Lucide props (absoluteStrokeWidth, etc.). */
  lucideProps?: Omit<LucideProps, 'size' | 'color' | 'strokeWidth' | 'fill'>;
};
