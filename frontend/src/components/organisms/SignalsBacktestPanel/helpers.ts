import type { ScannerBacktest, ScannerBacktestSignal, ScannerBacktestTrade } from '@/libs/api';

export function hasBacktestHitDetails(row: ScannerBacktestSignal): boolean {
  return (row.factors?.length ?? 0) > 0 || row.trade != null;
}

export function tradePnlTone(trade: ScannerBacktestTrade | undefined): 'success' | 'error' | 'secondary' {
  if (trade == null) return 'secondary';
  if (trade.outcome === 'open') return 'secondary';
  if (trade.pnlPct > 0) return 'success';
  if (trade.pnlPct < 0) return 'error';
  return 'secondary';
}

export function formatLeverage(lev: number | undefined): string {
  if (lev == null || !Number.isFinite(lev) || lev <= 0) return '—';
  return `${lev}x`;
}

export function isWatchlistBacktest(job: ScannerBacktest | null | undefined): boolean {
  if (job == null) return false;
  return job.scope === 'watchlist' || job.symbol === 'watchlist';
}

export function watchlistMarketCount(job: ScannerBacktest | null | undefined): number {
  return job?.markets?.length ?? 0;
}
