import type { WallScanHit } from '@/libs/api';
import type { WallRow } from '@/components/organisms/WallsTable';

function money(value: string | undefined): string {
  const n = Number(value);
  if (!value || Number.isNaN(n)) return '—';
  return n.toLocaleString('en-US', { maximumFractionDigits: 0 });
}

function distance(value: string | undefined): string {
  if (!value) return '—';
  return `${value}%`;
}

export function hitsToRows(
  hits: WallScanHit[] | undefined,
  kindLabel: (kind: string) => string,
  sideLabel: (side: string) => string,
  yes: string,
  no: string,
): WallRow[] {
  return (hits ?? []).map((hit) => ({
    exchange: hit.exchange || '',
    symbol: hit.symbol || '',
    side: sideLabel(hit.side || ''),
    price: hit.price || '—',
    notional: money(hit.notional),
    distance: distance(hit.distancePct),
    kind: kindLabel(hit.kind || ''),
    appears: hit.appearCount ? String(hit.appearCount) : '—',
    present: hit.present ? yes : no,
  }));
}
