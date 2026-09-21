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

export function formatVolGap(value: string | undefined): string {
  if (value == null || value === '') return '';
  const n = Number(value);
  if (!Number.isFinite(n)) return '';
  const sign = n > 0 ? '+' : '';
  return `${sign}${n.toFixed(1)}`;
}

function window24(data: MarketOptions | undefined) {
  return data?.history?.windows?.find((row) => row.window === '24h');
}

export function optionsStatValues(data: MarketOptions | undefined, asset: string) {
  const w = window24(data);
  return {
    oiCoin: formatCompactAsset(data?.openInterest, asset),
    oiUsd: formatCompactAmount(data?.openInterestUsd),
    oiChange24h: w?.oi?.changePct,
    volCoin: formatCompactAsset(data?.volume, asset),
    volUsd: formatCompactAmount(data?.volumeUsd),
    pcr: formatPcr(data?.putCallOi),
    pcrChange24h: w?.putCall?.changePct,
    lean: data?.lean ?? '',
    dvol: formatIv(data?.volCompare?.dvol ?? data?.dvol),
    dvolChange: data?.dvolChange24h ?? w?.dvol?.change,
    realized: formatIv(data?.volCompare?.realized),
    volGap: formatVolGap(data?.volCompare?.gap),
    atm: formatIv(data?.atm?.iv),
    atmDte: data?.atm?.daysToExpiry,
  };
}
