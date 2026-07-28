/**
 * Format a numeric or string price for display.
 * Tiny non-zero values use scientific notation instead of rounding to 0.
 */
export function formatPrice(value: string | number | null | undefined): string {
  if (value === null || value === undefined || value === '') {
    return '—';
  }
  const n = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(n)) {
    return '—';
  }
  if (n === 0) {
    return '0';
  }
  const abs = Math.abs(n);
  if (abs > 0 && abs < 1e-6) {
    return n.toExponential(2);
  }
  if (abs >= 1000) {
    return n.toLocaleString(undefined, { maximumFractionDigits: 2 });
  }
  if (abs >= 1) {
    return n.toLocaleString(undefined, { maximumFractionDigits: 4 });
  }
  return n.toLocaleString(undefined, { maximumFractionDigits: 8 });
}
