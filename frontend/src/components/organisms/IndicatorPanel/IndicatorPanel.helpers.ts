import { EMA_COLORS, FALLBACK_EMA_COLORS } from './IndicatorPanel.constants';

/** Color for EMA period key (known keys) or fallback palette by index. */
export function emaColor(key: string, index: number): string {
  return EMA_COLORS[key] ?? FALLBACK_EMA_COLORS[index % FALLBACK_EMA_COLORS.length]!;
}
