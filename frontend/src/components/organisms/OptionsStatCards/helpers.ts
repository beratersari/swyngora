import { formatCompactAmount, formatCompactAsset } from '@/libs/utils';
import type { MarketOptions } from '@/libs/api';

export function formatIv(value: string | undefined): string {
  if (value == null || value === '') return '—';
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  return `${n.toFixed(1)}%`;
}

export function formatPcr(value: string | undefined): string {
  if (value == null || value === '') return '—';
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  return n.toFixed(2);
}

export function optionsStatValues(data: MarketOptions | undefined, asset: string) {
  return {
    oiCoin: formatCompactAsset(data?.openInterest, asset),
    oiUsd: formatCompactAmount(data?.openInterestUsd),
    volCoin: formatCompactAsset(data?.volume, asset),
    volUsd: formatCompactAmount(data?.volumeUsd),
    pcr: formatPcr(data?.putCallOi),
    lean: data?.lean ?? '',
    dvol: formatIv(data?.dvol),
    dvolChange: data?.dvolChange24h,
    atm: formatIv(data?.atm?.iv),
    atmDte: data?.atm?.daysToExpiry,
  };
}
