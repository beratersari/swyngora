import type { TapeBacktestQuery } from '@/libs/api';
import { TAPE_DEFAULTS, TAPE_EXCHANGES, TAPE_SIDES, TAPE_THRESHOLD_BOUNDS } from './constants';
import type { TapeCondition, TapeDraft } from './TapeBacktestPanel.types';

export function isTapeCondition(value: string): value is TapeCondition {
  return (
    value === 'cvd' ||
    value === 'liquidation' ||
    value === 'book' ||
    value === 'funding' ||
    value === 'oi' ||
    value === 'gap' ||
    value === 'size'
  );
}

export type TapeSide = (typeof TAPE_SIDES)[TapeCondition][number];

export function tapeSides(condition: TapeCondition): readonly TapeSide[] {
  return TAPE_SIDES[condition];
}

export function tapeExchanges(condition: TapeCondition): readonly string[] {
  return TAPE_EXCHANGES[condition];
}

export function tapeDefaults(condition: TapeCondition): { side: string; threshold: number } {
  return TAPE_DEFAULTS[condition];
}

export function tapeThresholdBounds(condition: TapeCondition): { min: number; max: number; step: number } {
  return TAPE_THRESHOLD_BOUNDS[condition];
}

export function formatTapeReading(condition: string, value: number | null | undefined): string {
  if (value == null || Number.isNaN(value)) return '—';
  if (condition === 'liquidation') return `${value.toFixed(2)}×`;
  if (condition === 'funding') return `${value.toFixed(3)}%`;
  if (condition === 'oi') return `${value.toFixed(1)}%`;
  if (condition === 'gap') {
    const sign = value > 0 ? '+' : '';
    return `${sign}${value.toFixed(1)} pp`;
  }
  return `${(value * 100).toFixed(1)}%`;
}

export function tapeQueryFromDraft(draft: TapeDraft, now = Date.now()): TapeBacktestQuery {
  const symbol = draft.symbol.trim().toUpperCase();
  return {
    symbol,
    exchange: draft.exchange,
    condition: draft.condition,
    side: draft.side,
    threshold: draft.threshold,
    from: new Date(now - draft.days * 24 * 60 * 60 * 1000).toISOString(),
  };
}
