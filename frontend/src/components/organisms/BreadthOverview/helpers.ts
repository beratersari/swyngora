import type { MarketBreadthWindow } from '@/libs/api';

export function windowMix(w?: MarketBreadthWindow | null): string {
  if (!w?.counts) return '—';
  const up = w.counts.upPct ?? String(w.counts.up ?? '—');
  const down = w.counts.downPct ?? String(w.counts.down ?? '—');
  return `${up}% up · ${down}% down`;
}
