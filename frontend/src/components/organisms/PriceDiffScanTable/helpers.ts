import type { PriceDiffQuote } from '@/libs/api';

export function formatArbMoney(raw: string | number | undefined): string {
  if (raw == null || raw === '') return '—';
  const n = typeof raw === 'number' ? raw : Number(raw);
  if (!Number.isFinite(n)) return String(raw);
  const abs = Math.abs(n);
  const digits = abs >= 100 ? 2 : abs >= 1 ? 3 : 4;
  return n.toLocaleString(undefined, { maximumFractionDigits: digits });
}

export function formatArbPct(raw: number | string | undefined): string {
  if (raw == null || raw === '') return '—';
  const n = typeof raw === 'number' ? raw : Number(raw);
  if (!Number.isFinite(n)) return String(raw);
  return `${n.toFixed(3)}%`;
}

export function routeLabel(row: PriceDiffQuote): string {
  const buy = row.buyExchange ?? '—';
  const sell = row.sellExchange ?? '—';
  return `${buy} → ${sell}`;
}
