import type { BookGapScanHit } from '@/libs/api';
import type { GapRow } from '@/components/organisms/GapsTable';

function money(value: string | undefined): string {
  const n = Number(value);
  if (!value || Number.isNaN(n)) return '—';
  return n.toLocaleString('en-US', { maximumFractionDigits: 0 });
}

function pct(value: number | undefined): string {
  if (value === undefined || Number.isNaN(value)) return '—';
  return `${value.toFixed(2)}%`;
}

export function hitsToRows(
  hits: BookGapScanHit[] | undefined,
  exchange: string,
  easierLabel: (easier: string) => string,
): GapRow[] {
  return (hits ?? []).map((hit) => ({
    exchange: hit.scope && hit.scope !== 'all' ? hit.scope : exchange === 'all' ? 'binance' : exchange,
    symbol: hit.symbol || '',
    easier: easierLabel(hit.easier || 'even'),
    above: money(hit.upNotional),
    below: money(hit.downNotional),
    emptyAbove: pct(hit.upGapPct),
    emptyBelow: pct(hit.downGapPct),
    widest: pct(hit.widestPct),
  }));
}
