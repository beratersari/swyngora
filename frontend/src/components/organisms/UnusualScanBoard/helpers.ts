import { formatUnusualValue } from '@/components/organisms/UnusualContext/helpers';
import type { MarketUnusualScanHit } from '@/libs/api';

/** Spot route that opens the coin tape. OKX has no spot page, so that row opens Binance. */
export function unusualTapePath(exchange: string | undefined, symbol: string): string {
  const venue = exchange === 'bybit' ? 'bybit' : 'binance';
  return `/markets/${venue}/${encodeURIComponent(symbol)}?tab=tape`;
}

export function unusualHitKey(hit: MarketUnusualScanHit): string {
  return `${hit.symbol ?? ''}|${hit.exchange ?? ''}|${hit.metric ?? ''}`;
}

/** i18n key under the unusual namespace, plus the count that key interpolates. */
export function unusualSpanKey(spanHours: number | undefined): { key: string; count?: number } {
  const hours = spanHours ?? 0;
  if (hours < 1) return { key: 'span.underHour' };
  if (hours < 36) {
    const count = Math.max(1, Math.round(hours));
    return { key: count === 1 ? 'span.hour' : 'span.hours', count };
  }
  return { key: 'span.days', count: Math.max(2, Math.round(hours / 24)) };
}

export function unusualValue(hit: MarketUnusualScanHit): string {
  return formatUnusualValue(hit) ?? '—';
}
