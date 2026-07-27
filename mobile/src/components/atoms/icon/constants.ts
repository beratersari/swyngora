/** Default Lucide icon pixel sizes for product UI. */
export const ICON_SIZES = {
  xs: 14,
  sm: 16,
  md: 20,
  lg: 24,
  xl: 28,
} as const;

export type IconSizeName = keyof typeof ICON_SIZES;

/** Favorites gold — matches prior star affordance. */
export const ICON_FAVORITE_GOLD = '#F5C542';
