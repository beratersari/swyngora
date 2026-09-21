import type { SplitBarPart } from './SplitBar.types';

/** Width of each part as a percent string of the whole bar. */
export function splitWidths(parts: SplitBarPart[]): string[] {
  const weights = parts.map((part) => (Number.isFinite(part.pct) && part.pct > 0 ? part.pct : 0));
  const total = weights.reduce((sum, weight) => sum + weight, 0);
  if (total <= 0) return parts.map(() => '0%');
  return weights.map((weight) => `${(weight / total) * 100}%`);
}
