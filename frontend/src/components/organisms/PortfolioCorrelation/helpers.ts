import type { PortfolioCorrelation } from '@/libs/api';
import { SWING_EPSILON, TOGETHER_FOLLOWS, TOGETHER_INVERSE } from './constants';
import type { CorrelationKind } from './PortfolioCorrelation.types';

type Book = Pick<
  PortfolioCorrelation,
  'holdings' | 'together' | 'swingPct' | 'apartPct' | 'missing' | 'pairs'
>;

export function formatCorr(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '—';
  return value.toFixed(2);
}

export function formatSwing(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '—';
  return (Math.round(value * 10) / 10).toFixed(1);
}

export function correlationKind(report: Book | null | undefined): CorrelationKind {
  const holdings = report?.holdings ?? [];
  const swing = report?.swingPct;
  if (holdings.length === 0 && (report?.missing?.length ?? 0) === 0) return 'empty';
  if (swing == null || !Number.isFinite(swing)) return 'incomplete';
  if (holdings.length < 2 || report?.together == null) return 'one';
  const together = report.together;
  const apart = report.apartPct;
  if (together <= TOGETHER_INVERSE) return 'inverse';
  if (together >= TOGETHER_FOLLOWS && apart != null && swing + SWING_EPSILON < apart) return 'hedge';
  if (together >= TOGETHER_FOLLOWS) return 'together';
  return 'mixed';
}

export type SwingChange = 'up' | 'down' | 'same' | 'first' | 'unknown';

export function swingChange(
  before: number | null | undefined,
  after: number | null | undefined,
  complete: boolean | undefined,
): SwingChange {
  if (!complete || after == null || !Number.isFinite(after)) return 'unknown';
  if (before == null || !Number.isFinite(before)) return 'first';
  if (after > before + SWING_EPSILON) return 'up';
  if (after + SWING_EPSILON < before) return 'down';
  return 'same';
}
