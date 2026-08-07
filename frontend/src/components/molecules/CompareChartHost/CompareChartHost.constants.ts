import { palette } from '@/styles/tokens';

export const DEFAULT_HEIGHT = 360;

/** Compare overlays — brand greens only (no hard-coded off-palette hex). */
export const SERIES_COLORS = [
  palette.caribbeanGreen,
  palette.mountainMeadow,
  palette.mint,
] as const;
