import type { OrderBookLevel } from '@/libs/api';
import { MAX_BOOK_DECIMALS, MIN_QTY_DECIMALS } from './constants';

export function maxNotional(levels: OrderBookLevel[] | undefined): number {
  let max = 0;
  for (const lv of levels ?? []) {
    const n = Number.parseFloat(lv.notional ?? '0');
    if (Number.isFinite(n) && n > max) max = n;
  }
  return max;
}

export function depthPct(notional: string | undefined, max: number): number {
  if (max <= 0) return 0;
  const n = Number.parseFloat(notional ?? '0');
  if (!Number.isFinite(n) || n <= 0) return 0;
  return Math.min(100, (n / max) * 100);
}

/** Asks high→low so the spread sits in the middle of the ladder. */
export function asksHighToLow(asks: OrderBookLevel[] | undefined): OrderBookLevel[] {
  if (!asks?.length) return [];
  return [...asks].reverse();
}

/** Decimal places implied by a group step, e.g. "0.01" → 2, "1" → 0. */
export function priceDecimalsFromGroup(group: string | undefined): number {
  const raw = (group ?? '').trim();
  const n = Number.parseFloat(raw);
  if (!Number.isFinite(n) || n <= 0) return 2;
  if (n >= 1) return 0;
  const dot = raw.indexOf('.');
  if (dot < 0) return 0;
  const frac = raw.slice(dot + 1).replace(/0+$/, '');
  return Math.min(MAX_BOOK_DECIMALS, frac.length);
}

function fractionalDigits(value: string | undefined): number {
  const raw = (value ?? '').trim();
  if (!raw) return 0;
  const n = Number.parseFloat(raw);
  if (!Number.isFinite(n)) return 0;
  const normalized = n.toFixed(MAX_BOOK_DECIMALS).replace(/0+$/, '');
  const dot = normalized.indexOf('.');
  if (dot < 0) return 0;
  return normalized.length - dot - 1;
}

/** Shared qty/sum decimals for a ladder so figures share a decimal column. */
export function qtyDecimalsFromLevels(levels: OrderBookLevel[]): number {
  let max = MIN_QTY_DECIMALS;
  for (const lv of levels) {
    max = Math.max(max, fractionalDigits(lv.quantity), fractionalDigits(lv.cumulative));
  }
  return Math.min(MAX_BOOK_DECIMALS, max);
}

/** Fixed-scale amount so digits and the decimal point line up (tabular-nums). */
export function formatBookAmount(value: string | undefined, decimals: number): string {
  const n = Number.parseFloat(value ?? '');
  if (!Number.isFinite(n)) return '—';
  const d = Math.max(0, Math.min(MAX_BOOK_DECIMALS, decimals));
  return n.toFixed(d);
}

const TINY_PRICE = 0.001;

/** Widest decimal places among group steps so labels share one decimal column. */
export function groupLabelDecimals(steps: string[]): number {
  let max = 0;
  for (const step of steps) {
    max = Math.max(max, priceDecimalsFromGroup(step));
  }
  return Math.min(MAX_BOOK_DECIMALS, max);
}

/** Pad a group step to `decimals` so 0.0001 lines up with 0.000001. */
export function formatGroupLabel(group: string | undefined, decimals?: number): string {
  const raw = (group ?? '').trim();
  const n = Number.parseFloat(raw);
  if (!Number.isFinite(n) || n <= 0) return raw || '—';
  const d = decimals ?? priceDecimalsFromGroup(raw);
  return formatBookAmount(raw, d);
}

/** Shared 10^exp for a tiny book so every price uses the same suffix (1.20e-5). */
export function sharedPriceExponent(
  group: string | undefined,
  sample: string | undefined,
): number | null {
  const sampleN = Number.parseFloat(sample ?? '');
  const groupN = Number.parseFloat(group ?? '');
  const ref =
    Number.isFinite(sampleN) && sampleN !== 0
      ? Math.abs(sampleN)
      : Number.isFinite(groupN) && groupN > 0
        ? groupN
        : NaN;
  if (!Number.isFinite(ref) || ref >= TINY_PRICE) return null;
  return Math.floor(Math.log10(ref));
}

export function formatBookPrice(
  value: string | undefined,
  decimals: number,
  exp: number | null,
): string {
  const n = Number.parseFloat(value ?? '');
  if (!Number.isFinite(n)) return '—';
  if (exp == null || Math.abs(n) >= TINY_PRICE) {
    return formatBookAmount(value, decimals);
  }
  const mant = n / 10 ** exp;
  return `${mant.toFixed(2)}e${exp}`;
}

export function padLeft(value: string, width: number): string {
  if (value.length >= width) return value;
  return `${' '.repeat(width - value.length)}${value}`;
}

/**
 * Suggested steps only. A stale finer group (e.g. leftover 1e-8 on PIVX)
 * is dropped so it cannot stretch the picker.
 */
export function visibleGroupSteps(suggested: string[], active: string): string[] {
  const steps = suggested.filter((s) => s.trim().length > 0);
  if (!active || steps.includes(active)) return steps;
  const activeN = Number.parseFloat(active);
  const suggestedN = steps
    .map((s) => Number.parseFloat(s))
    .filter((n) => Number.isFinite(n) && n > 0);
  const floor = suggestedN.length ? Math.min(...suggestedN) : NaN;
  if (!Number.isFinite(activeN) || activeN <= 0) return steps;
  if (Number.isFinite(floor) && activeN + 1e-18 < floor) return steps;
  return [active, ...steps];
}

/** One markdown-table row: `| 0.000010 |    12 |    40 |` */
export function markdownBookRow(cells: string[], widths: number[]): string {
  const inner = cells.map((cell, i) => ` ${padLeft(cell, widths[i] ?? cell.length)} `).join('|');
  return `|${inner}|`;
}

export function markdownRule(widths: number[]): string {
  const inner = widths.map((w) => ` ${'-'.repeat(Math.max(1, w - 1))}: `).join('|');
  return `|${inner}|`;
}

export function columnWidths(rows: string[][]): number[] {
  const cols = Math.max(0, ...rows.map((r) => r.length));
  const widths = Array.from({ length: cols }, () => 0);
  for (const row of rows) {
    for (let i = 0; i < row.length; i += 1) {
      widths[i] = Math.max(widths[i], row[i]?.length ?? 0);
    }
  }
  return widths;
}
