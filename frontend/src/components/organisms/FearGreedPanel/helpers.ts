import type { FearGreedSample, FearGreedWindow } from '@/libs/api';
import { FEAR_GREED_HISTORY_BARS } from './constants';
import type { FearGreedTone } from './FearGreedPanel.types';

export function fearGreedTone(level?: string): FearGreedTone {
  switch (level) {
    case 'extreme_greed':
    case 'greed':
      return 'success';
    case 'extreme_fear':
    case 'fear':
      return 'error';
    case 'neutral':
      return 'warning';
    default:
      return 'secondary';
  }
}

export function recentFearGreed(history: FearGreedSample[] | undefined, limit = FEAR_GREED_HISTORY_BARS): FearGreedSample[] {
  const rows = history ?? [];
  if (rows.length <= limit) {
    return rows;
  }
  return rows.slice(rows.length - limit);
}

export function fearGreedWindowLine(w: FearGreedWindow | undefined): string | null {
  if (!w?.complete || w.pastScore == null || w.score == null) {
    return null;
  }
  const change = w.change ?? w.score - w.pastScore;
  const signed = change > 0 ? `+${change}` : String(change);
  return `${w.window} ${w.pastScore} → ${w.score} (${signed})`;
}
