import type { MarketLiquidationLevels } from '@/libs/api';
import type { LevelRow, TimeRow } from './LiquidationBarChart.types';

export function parseNotional(value: string | undefined): number {
  if (!value) return 0;
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function toLevelRows(data?: MarketLiquidationLevels | null): LevelRow[] {
  return (data?.levels ?? [])
    .map((lv) => ({
      price: parseNotional(lv.price),
      longN: parseNotional(lv.longNotional),
      shortN: parseNotional(lv.shortNotional),
      totalN: parseNotional(lv.totalNotional),
    }))
    .filter((row) => Number.isFinite(row.price) && row.price > 0);
}

export function toTimeRows(data?: MarketLiquidationLevels | null): TimeRow[] {
  return (data?.bars ?? []).map((bar) => {
    const t = Date.parse(bar.t ?? '');
    const d = Number.isFinite(t) ? new Date(t) : null;
    const label = d
      ? d.toISOString().slice(11, 16)
      : '';
    return {
      t: Number.isFinite(t) ? t : 0,
      label,
      longN: parseNotional(bar.longNotional),
      shortN: parseNotional(bar.shortNotional),
      totalN: parseNotional(bar.totalNotional),
    };
  });
}

export function maxSide(rows: { longN: number; shortN: number }[]): number {
  let m = 0;
  for (const row of rows) {
    m = Math.max(m, row.longN, row.shortN);
  }
  return m;
}

export function maxTotal(rows: { totalN: number }[]): number {
  let m = 0;
  for (const row of rows) {
    m = Math.max(m, row.totalN);
  }
  return m;
}

export function isLevelsKind(data?: MarketLiquidationLevels | null): boolean {
  return data?.kind === 'levels';
}
