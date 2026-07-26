import type { SpotMarket } from '@/libs/api';
import {
  changeTone,
  formatChangePercent,
  formatCompactUsd,
  formatPrice,
} from '@/libs/utils';
import type { MarketRowViewModel } from '@/components/organisms/MarketRow';

export function mapSpotMarketToRow(item: SpotMarket): MarketRowViewModel {
  const symbol = item.symbol ?? '—';
  return {
    id: symbol,
    symbol,
    lastPriceLabel: formatPrice(item.lastPrice),
    changePercentLabel: formatChangePercent(item.priceChangePercent),
    changeTone: changeTone(item.priceChangePercent),
    quoteVolumeLabel: formatCompactUsd(item.quoteVolume),
    marketCapLabel: formatCompactUsd(item.marketCapCirculating),
    tagsLabel: (item.tags ?? []).slice(0, 4).join(' · '),
  };
}

export function pageRangeLabel(offset: number, limit: number, total: number): string {
  if (total === 0) return '0 results';
  const start = offset + 1;
  const end = Math.min(offset + limit, total);
  return `${start}–${end} of ${total}`;
}
