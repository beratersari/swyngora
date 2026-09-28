/** Live total exposure, with the cap when one is set. */
export function exposureLine(pct?: number | null, cap?: number | null): string {
  if (pct == null || !Number.isFinite(pct)) return '';
  const now = trimPct(pct);
  if (cap == null || !Number.isFinite(cap) || cap <= 0) return `${now}%`;
  return `${now}% · ${trimPct(cap)}%`;
}

function trimPct(n: number): string {
  return n.toFixed(2).replace(/\.?0+$/, '');
}
