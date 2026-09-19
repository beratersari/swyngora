import type { BasisKind, BasisRicher, BasisShape, BasisSlope, BasisTrend } from './BasisPanel.types';

export function kindTone(kind: BasisKind | undefined): 'premium' | 'discount' | 'flat' {
  if (kind === 'premium' || kind === 'discount') return kind;
  return 'flat';
}

export function shapeTone(shape: BasisShape | undefined): 'contango' | 'backwardation' | 'flat' | 'humped' {
  if (shape === 'contango' || shape === 'backwardation' || shape === 'humped') return shape;
  return 'flat';
}

export function richerLine(richer: BasisRicher | undefined): 'richerFar' | 'richerNear' | 'richerFlat' {
  if (richer === 'far') return 'richerFar';
  if (richer === 'near') return 'richerNear';
  return 'richerFlat';
}

export function venueLabel(exchange: string | undefined): string {
  if (!exchange) return '';
  return exchange.charAt(0).toUpperCase() + exchange.slice(1);
}

export function trendLabel(trend: BasisTrend | undefined): BasisTrend {
  if (trend === 'expanding' || trend === 'shrinking' || trend === 'flipped') return trend;
  return 'stable';
}

export function slopeLabel(slope: BasisSlope | undefined): BasisSlope {
  if (slope === 'up' || slope === 'down') return slope;
  return 'flat';
}
