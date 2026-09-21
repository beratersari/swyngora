export function formatSignedTapeNum(raw: string | number | null | undefined): string {
  if (raw === null || raw === undefined || raw === '') return '—';
  const n = typeof raw === 'number' ? raw : Number(raw);
  if (!Number.isFinite(n)) return String(raw);
  const body = formatTapeNum(Math.abs(n));
  if (n > 0) return `+${body}`;
  if (n < 0) return `-${body}`;
  return body;
}

export function perpBidShareLine(w?: {
  complete?: boolean;
  pastBidPct?: string;
  bidPct?: string;
  changePct?: string;
} | null): string {
  if (!w?.complete || !w.pastBidPct || !w.bidPct) return '—';
  return `${w.pastBidPct}% → ${w.bidPct}%${w.changePct ? ` (${w.changePct})` : ''}`;
}

export function findPerpCoin<T extends { symbol?: string; base?: string }>(
  coins: T[] | undefined,
  symbol?: string | null,
): T | undefined {
  if (!coins?.length || !symbol) return undefined;
  const want = symbol.toUpperCase();
  const base = want.replace(/USDT$|USDC$|-USD$|-USDT$/, '');
  return (
    coins.find((c) => (c.symbol ?? '').toUpperCase() === want) ??
    coins.find((c) => (c.base ?? '').toUpperCase() === base)
  );
}

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

export function formatSignedFundingPct(raw: string | number | null | undefined): string {
  if (raw === null || raw === undefined || raw === '') return '—';
  const n = typeof raw === 'number' ? raw : Number(raw);
  if (!Number.isFinite(n)) return String(raw);
  const body = `${Math.abs(n).toFixed(4)}%`;
  if (n > 0) return `+${body}`;
  if (n < 0) return `-${body}`;
  return '0.0000%';
}

export function windowByName<T extends { window?: string }>(
  windows: T[] | undefined,
  name: string,
): T | undefined {
  return windows?.find((w) => w.window === name);
}
