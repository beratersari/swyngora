import { isCardWindow, type LiqCardWindowId } from '@/components/organisms/LiquidationWindowCards';
import { DEFAULT_LIQ_SYMBOL, DEFAULT_LIQ_WINDOW } from './LiquidationsPage.constants';

export type LiqPageView = 'overview' | 'heatmap' | 'chart';
export type LiqPageExchange = 'all' | 'binance' | 'bybit';
export type LiqChartRange = '12h' | '24h';

export function parseLiqView(raw: string | null): LiqPageView {
  if (raw === 'heatmap' || raw === 'chart') return raw;
  return 'overview';
}

export function parseLiqChartSymbol(raw: string | null): string {
  const s = (raw ?? '').trim().toUpperCase().replace(/-/g, '');
  if (s === 'ALL' || s === '*') return 'all';
  if (!s) return DEFAULT_LIQ_SYMBOL;
  const pair = parseLiqSymbol(s);
  if (pair.endsWith('USDT') || pair.endsWith('USDC')) return pair;
  return `${pair}USDT`;
}

export function parseLiqChartRange(raw: string | null): LiqChartRange {
  return raw === '12h' ? '12h' : '24h';
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
