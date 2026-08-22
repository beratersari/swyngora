export function formatTapeNum(raw: string | number | null | undefined): string {
  if (raw === null || raw === undefined || raw === '') return '—';
  const n = typeof raw === 'number' ? raw : Number(raw);
  if (!Number.isFinite(n)) return String(raw);
  const abs = Math.abs(n);
  const sign = n < 0 ? '-' : '';
  if (abs >= 1e12) return `${sign}${(abs / 1e12).toFixed(2)}T`;
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(2)}B`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(2)}K`;
  return n.toLocaleString(undefined, { maximumFractionDigits: 4 });
}

export function formatFundingPct(raw: string | number | null | undefined): string {
  if (raw === null || raw === undefined || raw === '') return '—';
  const n = typeof raw === 'number' ? raw : Number(raw);
  if (!Number.isFinite(n)) return String(raw);
  return `${n.toFixed(4)}%`;
}

export function windowByName<T extends { window?: string }>(
  windows: T[] | undefined,
  name: string,
): T | undefined {
  return windows?.find((w) => w.window === name);
}
