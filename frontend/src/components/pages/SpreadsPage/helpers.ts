import type { WideSpreadHit } from '@/libs/api';
import type { SpreadRow } from '@/components/organisms/SpreadsTable';

export function hitsToRows(hits: WideSpreadHit[] | undefined): SpreadRow[] {
  return (hits ?? [])
    .filter((hit) => hit.symbol && hit.exchange)
    .map((hit) => ({
      exchange: hit.exchange ?? '',
      symbol: hit.symbol ?? '',
      current: hit.currentPct ? `${hit.currentPct}%` : '—',
      typical: hit.typicalPct ? `${hit.typicalPct}%` : '—',
      ratio: hit.ratio ? `${hit.ratio}×` : '—',
    }));
}
