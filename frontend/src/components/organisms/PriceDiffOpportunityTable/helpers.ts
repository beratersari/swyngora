import type { TFunction } from 'i18next';
import type { PriceDiffOpportunity, PriceDiffOppTick } from '@/libs/api';
import { formatArbPct } from '@/components/organisms/PriceDiffScanTable';

export function formatOppDuration(sec: number | undefined): string {
  if (sec == null || !Number.isFinite(sec) || sec < 0) return '—';
  if (sec < 60) return `${Math.round(sec)}s`;
  if (sec < 3600) {
    const m = Math.floor(sec / 60);
    const s = Math.round(sec % 60);
    return s === 0 ? `${m}m` : `${m}m ${s}s`;
  }
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return m === 0 ? `${h}h` : `${h}h ${m}m`;
}

export function formatOppTime(raw: string | undefined): string {
  if (!raw) return '—';
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return raw;
  return d.toISOString().replace('T', ' ').replace(/\.\d+Z$/, ' UTC');
}

export function closeReasonLabel(t: TFunction<'arb'>, reason: string | undefined): string {
  switch (reason) {
    case 'below_min_profit':
      return t('price.closeReasons.belowMinProfit');
    case 'below_min_net':
      return t('price.closeReasons.belowMinNet');
    case 'thin_book':
      return t('price.closeReasons.thinBook');
    case 'paused':
      return t('price.closeReasons.paused');
    case 'venues_changed':
      return t('price.closeReasons.venuesChanged');
    default:
      return reason ? reason : t('price.closeReasons.unknown');
  }
}

export function oppRoute(row: PriceDiffOpportunity): string {
  return `${row.buyExchange ?? '—'} → ${row.sellExchange ?? '—'}`;
}

export function sparklinePoints(ticks: readonly PriceDiffOppTick[] | undefined, width = 160, height = 36): string {
  const vals = (ticks ?? [])
    .map((t) => t.netDiffPct)
    .filter((n): n is number => typeof n === 'number' && Number.isFinite(n));
  if (vals.length < 2) return '';
  const min = Math.min(...vals);
  const max = Math.max(...vals);
  const span = max - min || 1;
  return vals
    .map((v, i) => {
      const x = (i / (vals.length - 1)) * width;
      const y = height - ((v - min) / span) * (height - 4) - 2;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(' ');
}

export function lastNetLabel(row: PriceDiffOpportunity): string {
  return formatArbPct(row.review?.lastNetDiffPct ?? row.netDiffPct);
}
