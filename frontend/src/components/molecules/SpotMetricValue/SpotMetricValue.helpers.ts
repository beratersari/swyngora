import {
  formatChangePercent,
  formatCompactUsd,
  formatPrice,
  formatTradeCount,
  type SpotMetricDef,
} from '@/libs/utils';

/** Format a SpotMarket field value for the given metric definition. */
export function formatSpotMetricDisplay(
  format: SpotMetricDef['format'],
  raw: unknown,
  exchange?: string,
): string {
  switch (format) {
    case 'price':
      return formatPrice(raw as string | number | null | undefined);
    case 'changePercent':
      return formatChangePercent(raw as string | number | null | undefined);
    case 'compactUsd':
      return formatCompactUsd(raw as string | number | null | undefined);
    case 'tradeCount':
      return formatTradeCount(raw as number | null | undefined, exchange);
    case 'number':
      if (raw == null || raw === '') return '—';
      return typeof raw === 'number' ? raw.toLocaleString() : String(raw);
    case 'tags':
      return '—';
    default:
      return '—';
  }
}

export function asTagList(raw: unknown): string[] {
  return Array.isArray(raw) ? (raw as string[]) : [];
}
