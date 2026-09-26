import type { ImpactScanHit } from '@/libs/api';
import type { ImpactRow } from '@/components/organisms/ImpactTable';

function pct(value: number | undefined): string {
  if (value === undefined || Number.isNaN(value)) return '—';
  return `${value.toFixed(2)}%`;
}

export function hitsToRows(
  hits: ImpactScanHit[] | undefined,
  exchange: string,
  yes: string,
  no: string,
): ImpactRow[] {
  return (hits ?? []).map((hit) => ({
    exchange: hit.scope && hit.scope !== 'combined' ? hit.scope : exchange === 'all' ? 'binance' : exchange,
    symbol: hit.symbol || '',
    slippage: pct(hit.slippagePct),
    versusBest: pct(hit.slippageVsBestPct),
    impact: hit.impactAvailable ? pct(hit.impactPct) : '—',
    exhausted: hit.exhausted ? yes : no,
  }));
}
