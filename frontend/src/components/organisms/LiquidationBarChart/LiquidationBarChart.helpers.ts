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
      cumLong: parseNotional(lv.cumLong),
      cumShort: parseNotional(lv.cumShort),
      cumTotal: parseNotional(lv.cumTotal),
      byLeverage: (lv.byLeverage ?? [])
        .map((sl) => ({
          leverage: sl.leverage as 10 | 25 | 50 | 100,
          longN: parseNotional(sl.longNotional),
          shortN: parseNotional(sl.shortNotional),
        }))
        .filter((sl) => sl.leverage === 10 || sl.leverage === 25 || sl.leverage === 50 || sl.leverage === 100),
    }))
    .filter((row) => Number.isFinite(row.price) && row.price > 0);
}

export function sliceNotional(row: LevelRow, leverage: 10 | 25 | 50 | 100, side: 'long' | 'short'): number {
  const sl = row.byLeverage.find((s) => s.leverage === leverage);
  if (!sl) return 0;
  return side === 'long' ? sl.longN : sl.shortN;
}

export function barSide(row: LevelRow, lastPrice: number): 'long' | 'short' {
  if (lastPrice > 0 && row.price >= lastPrice) return 'short';
  if (lastPrice > 0) return 'long';
  return row.shortN >= row.longN ? 'short' : 'long';
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
