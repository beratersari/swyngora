import { formatDateTime, formatPrice } from '@/libs/utils';

type Translate = (key: string, options?: Record<string, unknown>) => string;

const KIND_KEYS: Record<string, Record<string, string>> = {
  priceCvd: {
    price_up_cvd_down: 'situationRecord.kinds.priceCvd_price_up_cvd_down',
    price_down_cvd_up: 'situationRecord.kinds.priceCvd_price_down_cvd_up',
    agrees: 'situationRecord.kinds.priceCvd_agrees',
  },
  openInterest: {
    opened: 'situationRecord.kinds.openInterest_opened',
    closed: 'situationRecord.kinds.openInterest_closed',
    flat: 'situationRecord.kinds.openInterest_flat',
  },
  liquidation: {
    long: 'situationRecord.kinds.liquidation_long',
    short: 'situationRecord.kinds.liquidation_short',
    none: 'situationRecord.kinds.liquidation_none',
  },
  prints: {
    long: 'situationRecord.kinds.prints_long',
    short: 'situationRecord.kinds.prints_short',
    even: 'situationRecord.kinds.prints_even',
    quiet: 'situationRecord.kinds.prints_quiet',
  },
  funding: {
    long: 'situationRecord.kinds.funding_long',
    short: 'situationRecord.kinds.funding_short',
    none: 'situationRecord.kinds.funding_none',
  },
  crowd: {
    top_more_long: 'situationRecord.kinds.crowd_top_more_long',
    top_more_short: 'situationRecord.kinds.crowd_top_more_short',
    even: 'situationRecord.kinds.crowd_even',
  },
  size: {
    whale_buy_retail_sell: 'situationRecord.kinds.size_whale_buy_retail_sell',
    whale_sell_retail_buy: 'situationRecord.kinds.size_whale_sell_retail_buy',
    both_buy: 'situationRecord.kinds.size_both_buy',
    both_sell: 'situationRecord.kinds.size_both_sell',
    whale_buy: 'situationRecord.kinds.size_whale_buy',
    whale_sell: 'situationRecord.kinds.size_whale_sell',
    retail_buy: 'situationRecord.kinds.size_retail_buy',
    retail_sell: 'situationRecord.kinds.size_retail_sell',
    flat: 'situationRecord.kinds.size_flat',
  },
};

export function situationRecordKindLabel(clause: string, kind: string, t: Translate): string {
  const key = KIND_KEYS[clause]?.[kind];
  if (!key) return kind;
  return t(key);
}

export function situationRecordRowKey(clause: string, kind: string): string {
  return `${clause}:${kind}`;
}

export type RecordClose = {
  exchange?: string;
  symbol?: string;
  closedAt?: string;
  realizedPnL?: number;
};

/** One sold slice on an opened row: symbol, venue, sell time, and that slice's profit. */
export function situationRecordCloseLine(
  close: RecordClose,
  t: Translate,
  locale?: string,
): string {
  const symbol = close.symbol?.trim() || '—';
  const exchange = close.exchange?.trim() || '';
  const time = formatDateTime(close.closedAt, locale);
  const pnl = formatRecordPnl(close.realizedPnL);
  if (!exchange) return t('situationRecord.closeNoVenue', { symbol, time, pnl });
  return t('situationRecord.close', { symbol, exchange, time, pnl });
}

/** Signed price text. A gain keeps the plus so a loss and a gain do not look the same. */
export function formatRecordPnl(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '—';
  const text = formatPrice(value);
  if (value > 0 && text !== '—' && !text.startsWith('+')) return `+${text}`;
  return text;
}
