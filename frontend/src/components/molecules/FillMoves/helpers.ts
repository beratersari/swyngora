import type { TFunction } from 'i18next';
import type { TextColor } from '@/styles/tokens';
import { changeTone, formatChangePercent } from '@/libs/utils';
import type { FillMove } from './FillMoves.types';

/** One window line. A ready row with no percent is shown blank, not as 0%. */
export function fillMoveLine(move: FillMove, t: TFunction<'portfolio'>): string {
  const window = move.window ?? '';
  if (move.status === 'ready' && move.pct != null && Number.isFinite(move.pct)) {
    return t('moves.ready', { window, pct: formatChangePercent(move.pct) });
  }
  if (move.status === 'pending') {
    return t('moves.pending', { window });
  }
  return t('moves.missing', { window });
}

export function fillMoveTone(move: FillMove): TextColor {
  if (move.status !== 'ready' || move.pct == null || !Number.isFinite(move.pct)) {
    return 'secondary';
  }
  return changeTone(move.pct);
}
