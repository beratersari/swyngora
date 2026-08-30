import type { LiquidationCardWindow } from './LiquidationWindowCards.types';
import { LIQ_CARD_WINDOWS, type LiqCardWindowId } from './LiquidationWindowCards.constants';

export function parseNotional(value: string | undefined): number {
  if (!value) return 0;
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function windowById(
  windows: LiquidationCardWindow[],
  id: string,
): LiquidationCardWindow | undefined {
  return windows.find((w) => w.window === id);
}

export function orderedCardWindows(windows: LiquidationCardWindow[]): LiquidationCardWindow[] {
  return LIQ_CARD_WINDOWS.map((id) => windowById(windows, id) ?? { window: id });
}

export function isCardWindow(value: string | null | undefined): value is LiqCardWindowId {
  return Boolean(value && (LIQ_CARD_WINDOWS as readonly string[]).includes(value));
}

export function longShare(row: LiquidationCardWindow | undefined): number {
  const longN = parseNotional(row?.longNotional);
  const shortN = parseNotional(row?.shortNotional);
  const total = longN + shortN;
  if (total <= 0) return 0.5;
  return longN / total;
}
