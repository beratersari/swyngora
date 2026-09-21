import type { MarketPerpDepthOverview } from '@/libs/api';
import { PERP_LIQUIDITY_LEADERS } from './constants';

export function leaderCoins(data?: MarketPerpDepthOverview | null) {
  const coins = data?.coins ?? [];
  return coins.filter((c) => !c.error && Number(c.total) > 0).slice(0, PERP_LIQUIDITY_LEADERS);
}

export function shareLabel(pct?: string | null): string {
  if (!pct) return '—';
  return `${pct}%`;
}
