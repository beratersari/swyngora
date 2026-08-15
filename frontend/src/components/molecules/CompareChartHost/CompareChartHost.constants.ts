import { palette } from '@/styles/tokens';

export const DEFAULT_HEIGHT = 360;

/** Compare overlays — amber / cobalt / up (no off-palette hex). */
export const SERIES_COLORS = [
  palette.amber,
  palette.cobalt,
  palette.up,
] as const;
