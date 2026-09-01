import type { LiqHeatRange } from './LiquidationHeatmap.types';

export const LIQ_HEATMAP_POLL_MS = 60_000;

export const LIQ_HEATMAP_RANGES: readonly LiqHeatRange[] = ['12h', '24h', '3d', '7d'];

export const DEFAULT_LIQ_HEATMAP_RANGE: LiqHeatRange = '24h';

export const HEATMAP_PAD = {
  left: 76,
  right: 18,
  top: 10,
  bottom: 26,
} as const;

export const PLOT_BG = '#0B1220';
export const AXIS_INK = '#9AA6B8';
export const GRID_LINE = 'rgba(255, 255, 255, 0.06)';
export const LAST_STROKE = '#E8EDF5';

export const HEAT_BED = { r: 11, g: 18, b: 32 };
export const HEAT_LOW = { r: 62, g: 48, b: 12 };
export const HEAT_MID = { r: 232, g: 140, b: 18 };
export const HEAT_HIGH = { r: 236, g: 48, b: 72 };
export const HEAT_PEAK = { r: 255, g: 92, b: 214 };

export const LONG_LOW = { r: 72, g: 22, b: 22 };
export const LONG_HIGH = { r: 234, g: 57, b: 67 };
export const SHORT_LOW = { r: 10, g: 56, b: 42 };
export const SHORT_HIGH = { r: 22, g: 199, b: 132 };
