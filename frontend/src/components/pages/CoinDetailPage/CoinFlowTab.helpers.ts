import type { MarketDeskReport } from '@/libs/api';
import { formatCompactAmount } from '@/libs/utils';

export function reportSummary(r?: MarketDeskReport | null): string {
  if (!r) return '';
  return (
    r.summary ||
    r.combined?.summary ||
    r.venues?.find((v) => v.summary)?.summary ||
    r.windows?.find((w) => w.summary)?.summary ||
    ''
  );
}

export function perpStats(r?: MarketDeskReport | null): { label: string; value: string }[] {
  const h = r?.combined?.headline;
  if (!h) return [];
  return [
    { label: 'Bid', value: formatCompactAmount(h.below?.notional) },
    { label: 'Ask', value: formatCompactAmount(h.above?.notional) },
    { label: 'Lean', value: h.lean || r?.combined?.lean || '—' },
  ];
}

export function surgeStats(r?: MarketDeskReport | null): { label: string; value: string }[] {
  const venue = r?.combined ?? r?.venues?.[0];
  const win = venue?.windows?.find((x) => x.window === '15m') ?? venue?.windows?.[0];
  if (!win) return [];
  return [
    { label: win.window || '15m', value: win.ratio ? `${win.ratio}×` : '—' },
    { label: 'Grade', value: win.grade || '—' },
  ];
}
