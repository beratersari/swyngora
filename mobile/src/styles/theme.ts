import { colors, semanticColors, spacing, radii, typeScale, fontFamilies } from './tokens';

export const theme = {
  colors,
  semantic: semanticColors,
  spacing,
  radii,
  typeScale,
  fontFamilies,
} as const;

export type AppTheme = typeof theme;
