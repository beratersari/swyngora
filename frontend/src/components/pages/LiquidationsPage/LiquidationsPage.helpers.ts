import { isCardWindow, type LiqCardWindowId } from '@/components/organisms/LiquidationWindowCards';
import { DEFAULT_LIQ_SYMBOL, DEFAULT_LIQ_WINDOW } from './LiquidationsPage.constants';

export type LiqPageView = 'overview' | 'heatmap';
export type LiqPageExchange = 'all' | 'binance' | 'bybit';

export function parseLiqView(raw: string | null): LiqPageView {
  return raw === 'heatmap' ? 'heatmap' : 'overview';
}

export function parseLiqWindow(raw: string | null): LiqCardWindowId {
  return isCardWindow(raw) ? raw : DEFAULT_LIQ_WINDOW;
}

export function parseLiqExchange(raw: string | null): LiqPageExchange {
  if (raw === 'binance' || raw === 'bybit' || raw === 'all') return raw;
  return 'all';
}

export function parseLiqSymbol(raw: string | null): string {
  const s = (raw ?? '').trim().toUpperCase().replace(/-/g, '');
  if (!s) return DEFAULT_LIQ_SYMBOL;
  if (s.endsWith('USDT') || s.endsWith('USDC')) return s;
  if (s.endsWith('USD')) return `${s.slice(0, -3)}USDT`;
  return s;
}
