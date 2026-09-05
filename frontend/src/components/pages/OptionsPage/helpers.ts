import { OPTIONS_ASSETS, type OptionsAsset } from './constants';

export function parseOptionsAsset(raw: string | null | undefined): OptionsAsset {
  const s = (raw ?? '').trim().toUpperCase();
  if (s === 'ETH' || s.startsWith('ETH')) return 'ETH';
  if ((OPTIONS_ASSETS as readonly string[]).includes(s) || s.startsWith('BTC')) return 'BTC';
  return 'BTC';
}
