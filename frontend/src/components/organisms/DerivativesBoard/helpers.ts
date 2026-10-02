import { formatCompactAmount, formatPrice } from '@/libs/utils';
import type { DerivativesCoin } from '@/libs/api';

const DASH = '—';

export function coinChartPath(exchange: string, symbol: string): string {
  const venue = !exchange || exchange === 'all' ? 'binance' : exchange;
  return `/markets/${venue}/${symbol}`;
}

export function definiteText(definite: boolean | undefined, value: string | undefined, format: (value: string) => string): string {
  if (!definite || value == null || value === '') return DASH;
  return format(value);
}

export function plainAmount(value: string): string {
  return formatCompactAmount(value);
}

export function signedAmount(value: string): string {
  const n = Number(value);
  if (!Number.isFinite(n)) return DASH;
  const body = formatCompactAmount(Math.abs(n));
  if (n > 0) return `+${body}`;
  if (n < 0) return `-${body}`;
  return body;
}

export function priceText(coin: DerivativesCoin): string {
  if (!coin.price) return DASH;
  return formatPrice(coin.price);
}

export function fundingText(coin: DerivativesCoin): string {
  if (!coin.fundingDefinite || !coin.fundingRatePct) return DASH;
  return `${coin.fundingRatePct}%`;
}
