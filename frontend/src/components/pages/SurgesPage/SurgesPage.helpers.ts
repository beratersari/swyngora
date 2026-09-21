import type { MarketVolumeSurgeHit } from '@/libs/api';
import type { VolumeSurgeRow } from '@/components/organisms/VolumeSurgeTable';

export function surgeHitsToRows(hits?: MarketVolumeSurgeHit[] | null): VolumeSurgeRow[] {
  return (hits ?? [])
    .filter((h) => h.symbol)
    .map((h) => ({
      exchange: (h.exchange || 'binance').toLowerCase(),
      symbol: String(h.symbol),
      maxRatio: h.maxRatio || '',
      hottest: h.hottest || '',
      grade: h.grade || '',
      dominant: h.dominant || '',
      summary: h.summary || '',
    }));
}
