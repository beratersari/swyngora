import type { MarketOptionsExpiry } from '@/libs/api';
import { expiryRowKey } from '@/components/organisms/OptionsExpiryBars/helpers';
import { OPTIONS_ASSETS, type OptionsAsset } from './constants';

export function parseOptionsAsset(raw: string | null | undefined): OptionsAsset {
  const s = (raw ?? '').trim().toUpperCase();
  if (s === 'ETH' || s.startsWith('ETH')) return 'ETH';
  if ((OPTIONS_ASSETS as readonly string[]).includes(s) || s.startsWith('BTC')) return 'BTC';
  return 'BTC';
}

export function parseSelectedExpiry(
  raw: string | null | undefined,
  expiries: MarketOptionsExpiry[] | undefined,
): string {
  const rows = expiries ?? [];
  if (rows.length === 0) return '';
  const want = (raw ?? '').trim();
  if (want) {
    const hit = rows.find((row) => {
      const key = expiryRowKey(row);
      return key === want || row.expiry === want || row.label === want;
    });
    if (hit) return expiryRowKey(hit);
  }
  const live = rows.find((row) => (row.daysToExpiry ?? 0) >= 0);
  return expiryRowKey(live ?? rows[0]);
}
