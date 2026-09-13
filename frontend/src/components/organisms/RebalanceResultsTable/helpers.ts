import type { RebalanceResult } from '@/libs/api';

export function formatRebalanceWhen(raw: string | undefined): string {
  if (!raw) return '—';
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return raw;
  return d.toISOString().replace('T', ' ').replace(/\.\d+Z$/, ' UTC');
}

export function formatRebalancePct(n: number | undefined): string {
  if (n == null || !Number.isFinite(n)) return '—';
  return `${n.toFixed(2)}%`;
}

export function formatRebalanceMoney(n: number | undefined, currency?: string): string {
  if (n == null || !Number.isFinite(n)) return '—';
  const abs = Math.abs(n);
  const digits = abs >= 100 ? 2 : abs >= 1 ? 3 : 4;
  const s = n.toLocaleString(undefined, { maximumFractionDigits: digits });
  return currency ? `${s} ${currency}` : s;
}

export function rebalanceStatusKey(status: RebalanceResult['status']): string {
  switch (status) {
    case 'completed':
      return 'rebalance.statusCompleted';
    case 'partial':
      return 'rebalance.statusPartial';
    case 'failed':
      return 'rebalance.statusFailed';
    case 'no_trades':
      return 'rebalance.statusNone';
    case 'interrupted':
      return 'rebalance.statusInterrupted';
    case 'running':
      return 'rebalance.statusRunning';
    default:
      return 'rebalance.statusUnknown';
  }
}

export function legStatusKey(status: string | undefined): string {
  if (status === 'filled') return 'rebalance.legFilled';
  if (status === 'failed') return 'rebalance.legFailed';
  return 'rebalance.legUnknown';
}
