import type { TakeProfitTargetInput } from '@/libs/hooks/orderPreview';

export type { TakeProfitTargetInput };

export type TargetRow = {
  percent: number | null;
  offsetPercent: number | null;
  price: number | null;
};

export const MAX_TAKE_PROFIT_TARGETS = 8;

const SUM_EPSILON = 0.01;

function positive(value: number | null | undefined): value is number {
  return value != null && Number.isFinite(value) && value > 0;
}

/** The 25% / 25% / 50% scale-out at +5%, +10%, and +15%. */
export function exampleScaleOut(): TargetRow[] {
  return [
    { percent: 25, offsetPercent: 5, price: null },
    { percent: 25, offsetPercent: 10, price: null },
    { percent: 50, offsetPercent: 15, price: null },
  ];
}

export function emptyTargetRow(): TargetRow {
  return { percent: null, offsetPercent: null, price: null };
}

export function targetShareSum(rows: TargetRow[]): number {
  return rows.reduce((sum, row) => sum + (positive(row.percent) ? row.percent : 0), 0);
}

/** True when every row has a share and either a gain or a price, and the shares add up to 100. */
export function targetsComplete(rows: TargetRow[]): boolean {
  if (rows.length === 0 || rows.length > MAX_TAKE_PROFIT_TARGETS) return false;
  if (Math.abs(targetShareSum(rows) - 100) > SUM_EPSILON) return false;
  return rows.every((row) => {
    const gain = positive(row.offsetPercent);
    const price = positive(row.price);
    return positive(row.percent) && gain !== price;
  });
}

/**
 * One absolute target for the whole size stays on takeProfitPrice.
 * Anything else is sent as takeProfitTargets.
 */
export function targetWire(rows: TargetRow[]): {
  takeProfitPrice?: number;
  takeProfitTargets?: TakeProfitTargetInput[];
} {
  if (
    rows.length === 1 &&
    rows[0].percent === 100 &&
    positive(rows[0].price) &&
    !positive(rows[0].offsetPercent)
  ) {
    return { takeProfitPrice: rows[0].price };
  }
  const takeProfitTargets = rows.map((row) => {
    const target: TakeProfitTargetInput = { percent: row.percent ?? 0 };
    if (positive(row.offsetPercent)) target.offsetPercent = row.offsetPercent;
    else if (positive(row.price)) target.price = row.price;
    return target;
  });
  return { takeProfitTargets };
}
