import {
  colors,
  palette,
  semanticColors,
  fontFamilies,
  fontWeights,
  typeScale,
  spacing,
  radii,
} from '@/styles/tokens';

/** App theme for styled-components ThemeProvider */
export const appTheme = {
  colors,
  palette,
  semantic: semanticColors,
  fontFamilies,
  fontWeights,
  typeScale,
  spacing,
  radii,
} as const;

export type AppTheme = typeof appTheme;
