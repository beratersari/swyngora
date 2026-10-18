/** Today's trading result, with a sign. Negative is a loss. */
export function lossLine(pct: number | null | undefined): string {
  if (pct == null || !Number.isFinite(pct)) return '—';
  const body = `${Math.abs(pct).toFixed(2)}%`;
  if (pct > 0) return `+${body}`;
  if (pct < 0) return `-${body}`;
  return body;
}
