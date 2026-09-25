import type { MarketOptionsStrikeOIRow, MarketOptionsStrikeOIWindow } from '@/libs/api';

export function windowRows(
  windows: MarketOptionsStrikeOIWindow[] | undefined,
  windowId: string,
  expiry: string | undefined,
): MarketOptionsStrikeOIRow[] {
  const win = (windows ?? []).find((row) => row.window === windowId && row.complete);
  if (!win || !expiry) return [];
  const want = expiry.trim();
  return (win.strikes ?? []).filter((row) => sameExpiry(row.expiry, want));
}

function sameExpiry(row: string | undefined, selected: string): boolean {
  const got = (row ?? '').trim();
  if (got === selected) return true;
  const a = Date.parse(got);
  const b = Date.parse(selected);
  return Number.isFinite(a) && a === b;
}

export function maxAbsChange(rows: MarketOptionsStrikeOIRow[]): number {
  let peak = 0;
  for (const row of rows) {
    const call = Math.abs(Number(row.callChange));
    const put = Math.abs(Number(row.putChange));
    if (Number.isFinite(call) && call > peak) peak = call;
    if (Number.isFinite(put) && put > peak) peak = put;
  }
  return peak;
}

export function changeBarPct(raw: string | undefined, peak: number): number {
  if (peak <= 0) return 0;
  const n = Math.abs(Number(raw));
  if (!Number.isFinite(n) || n <= 0) return 0;
  return Math.max(4, Math.min(100, (n / peak) * 100));
}

export function signedChange(raw: string | undefined): number {
  const n = Number(raw);
  return Number.isFinite(n) ? n : 0;
}
