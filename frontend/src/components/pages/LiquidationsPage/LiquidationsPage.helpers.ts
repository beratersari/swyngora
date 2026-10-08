import { LIQ_CHART_RANGES } from '@/components/organisms/LiquidationBarChart/LiquidationBarChart.constants';
import type { HuntPanel } from '@/components/organisms/LiquidationHunt/LiquidationHunt.types';
import { isCardWindow, type LiqCardWindowId } from '@/components/organisms/LiquidationWindowCards';
import { DEFAULT_LIQ_SYMBOL, DEFAULT_LIQ_WINDOW } from './LiquidationsPage.constants';

export type LiqPageView = 'overview' | 'heatmap' | 'open' | 'chart' | 'cascade' | 'hunt' | 'max-pain';
export type LiqPageExchange = 'all' | 'binance' | 'bybit' | 'okx';
export type LiqChartRange = (typeof LIQ_CHART_RANGES)[number];
export type LiqHuntPanel = HuntPanel;

export function parseLiqView(raw: string | null): LiqPageView {
  if (
    raw === 'overview' ||
    raw === 'heatmap' ||
    raw === 'open' ||
    raw === 'chart' ||
    raw === 'cascade' ||
    raw === 'hunt' ||
    raw === 'max-pain'
  ) {
    return raw;
  }
  return 'heatmap';
}

export function parseLiqHuntPanel(raw: string | null): LiqHuntPanel {
  return raw === 'path' ? 'path' : 'compare';
}

export function parseLiqCascadeSymbol(raw: string | null): string {
  const s = (raw ?? '').trim().toUpperCase().replace(/-/g, '');
  if (!s || s === 'ALL' || s === '*') return 'all';
  return parseLiqChartSymbol(s);
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
  if (raw && (LIQ_CHART_RANGES as readonly string[]).includes(raw)) return raw as LiqChartRange;
  return '24h';
}

export function parseLiqWindow(raw: string | null): LiqCardWindowId {
  return isCardWindow(raw) ? raw : DEFAULT_LIQ_WINDOW;
}

export function parseLiqExchange(raw: string | null): LiqPageExchange {
  if (raw === 'binance' || raw === 'bybit' || raw === 'okx' || raw === 'all') return raw;
  return 'all';
}

export function hasLiquidationTotals(
  overview:
    | {
        coins?: unknown[] | null;
        windows?: { totalNotional?: string | null; count?: number | null }[] | null;
      }
    | null
    | undefined,
): boolean {
  if (!overview) return false;
  if ((overview.coins?.length ?? 0) > 0) return true;
  return (overview.windows ?? []).some((row) => {
    const total = Number(row.totalNotional);
    return (Number.isFinite(total) && total > 0) || (row.count ?? 0) > 0;
  });
}

export function parseLiqSymbol(raw: string | null): string {
  const s = (raw ?? '').trim().toUpperCase().replace(/-/g, '');
  if (!s) return DEFAULT_LIQ_SYMBOL;
  if (s.endsWith('USDT') || s.endsWith('USDC')) return s;
  if (s.endsWith('USD')) return `${s.slice(0, -3)}USDT`;
  return s;
}
