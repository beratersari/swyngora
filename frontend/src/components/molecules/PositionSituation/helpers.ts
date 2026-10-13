import type { CoinSituation, SpotPosition } from '@/libs/api';
import type { SituationComparison } from '@/components/molecules/CoinSituation';
import { formatDateTime } from '@/libs/utils/formatMarket';
import { formatPrice } from '@/libs/utils/formatPrice';

const CASH_EQUITIES = new Set(['nasdaq', 'bist']);

export function isCashEquity(exchange: string | undefined): boolean {
  return CASH_EQUITIES.has((exchange ?? '').trim().toLowerCase());
}

export type OpenLotReadView = {
  id?: string;
  quantity?: number;
  openedAt?: string;
  sourceTradeId?: string;
  situation?: CoinSituation | null;
  newest?: boolean;
  comparison?: SituationComparison | null;
};

type LotSource = {
  openReads?: OpenLotReadView[] | null;
};

type ChangeSource = {
  liveOk?: boolean;
  openReads?: OpenLotReadView[] | null;
};

/** Remaining buys for the position list. The live comparison wins when it loaded. */
export function positionLotReads(
  position: LotSource | null | undefined,
  change: ChangeSource | null | undefined,
): OpenLotReadView[] | undefined {
  const fromChange = (change?.openReads ?? []).filter((lot) => (lot.quantity ?? 0) > 0);
  if (fromChange.length > 0) {
    if (change?.liveOk) return fromChange;
    return fromChange.map((lot) => ({ ...lot, comparison: undefined }));
  }
  const fromPosition = (position?.openReads ?? []).filter((lot) => (lot.quantity ?? 0) > 0);
  return fromPosition.length > 0 ? fromPosition : undefined;
}

type LotCaption = (key: string, values?: Record<string, string>) => string;

/** Size and fill time for one remaining buy. */
export function openLotCaption(
  quantity: number | undefined,
  openedAt: string | undefined,
  locale: string | undefined,
  t: LotCaption,
): string {
  const qty = formatPrice(quantity);
  const time = openedAt ? formatDateTime(openedAt, locale) : '';
  if (time && time !== '—') return t('detail:situation.lot', { qty, time });
  return t('detail:situation.lotQty', { qty });
}

/** The open spot row for this coin, when the selected book still holds it. */
export function openPositionFor(
  positions: SpotPosition[] | undefined,
  exchange: string,
  symbol: string,
): SpotPosition | undefined {
  return positions?.find(
    (row) => row.exchange === exchange && row.symbol === symbol && (row.quantity ?? 0) > 0,
  );
}
