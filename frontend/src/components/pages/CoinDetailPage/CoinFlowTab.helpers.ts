import type { MarketDeskReport, SpotPerpLiquidity } from '@/libs/api';
import { formatCompactAmount } from '@/libs/utils';

export type SpotPerpFact = { key: string; value: string };

export function spotPerpFacts(r?: SpotPerpLiquidity | null): SpotPerpFact[] {
  if (!r?.spot && !r?.perp) return [];
  return [
    { key: 'spotBid', value: formatCompactAmount(r.spot?.bidNotional) },
    { key: 'spotAsk', value: formatCompactAmount(r.spot?.askNotional) },
    { key: 'perpBid', value: formatCompactAmount(r.perp?.bidNotional) },
    { key: 'perpAsk', value: formatCompactAmount(r.perp?.askNotional) },
    { key: 'thinnerBook', value: r.thinnerBook || '—' },
    { key: 'spotSide', value: r.spot?.thinnerSide || '—' },
    { key: 'perpSide', value: r.perp?.thinnerSide || '—' },
  ];
}

export function spotPerpVenueLines(r?: SpotPerpLiquidity | null): string[] {
  return (r?.venues ?? [])
    .filter((v) => v.exchange)
    .map((v) => {
      if (!v.complete) return `${v.exchange}: ${v.error || '—'}`;
      return `${v.exchange}: spot ${formatCompactAmount(v.spotBid)} / ${formatCompactAmount(v.spotAsk)} · perp ${formatCompactAmount(v.perpBid)} / ${formatCompactAmount(v.perpAsk)} · ${v.thinnerBook || '—'}`;
    });
}

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
  const rows: { label: string; value: string }[] = [];
  if (h) {
    rows.push(
      { label: 'Bid', value: formatCompactAmount(h.below?.notional) },
      { label: 'Ask', value: formatCompactAmount(h.above?.notional) },
      { label: 'Lean', value: h.lean || r?.combined?.lean || '—' },
    );
  }
  for (const v of r?.venues ?? []) {
    if (!v.exchange || !v.headline) continue;
    rows.push({
      label: v.exchange,
      value: `${formatCompactAmount(v.headline.below?.notional)} / ${formatCompactAmount(v.headline.above?.notional)}`,
    });
  }
  return rows;
}

export function venueLines(r?: MarketDeskReport | null): { label: string; value: string }[] {
  return (r?.venues ?? [])
    .filter((v) => v.exchange)
    .map((v) => ({ label: v.exchange === 'okx' ? 'OKX' : (v.exchange ?? ''), value: v.summary || v.error || '—' }));
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
