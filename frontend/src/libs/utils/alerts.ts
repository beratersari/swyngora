/** Alert display helpers. Fire/edge evaluation lives in the Go API. */

export type AlertKind =
  | 'price_above'
  | 'price_below'
  | 'change_pct_above'
  | 'change_pct_below';

export function parseFiniteNumber(value: string | number | null | undefined): number | null {
  if (value === null || value === undefined || value === '') return null;
  const n = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(n) ? n : null;
}

export const ALERT_FIRE_COOLDOWN_MS = 5 * 60_000;

export function alertDisplayLabel(alert: {
  kind?: AlertKind;
  threshold?: number;
  symbol?: string;
}): string {
  const thr = alert.threshold ?? 0;
  switch (alert.kind) {
    case 'price_above':
      return `${alert.symbol} ≥ ${thr}`;
    case 'price_below':
      return `${alert.symbol} ≤ ${thr}`;
    case 'change_pct_above':
      return `${alert.symbol} 24h ≥ ${thr}%`;
    case 'change_pct_below':
      return `${alert.symbol} 24h ≤ ${thr}%`;
    default:
      return alert.symbol ?? 'alert';
  }
}

export const ALERT_KINDS: AlertKind[] = [
  'price_above',
  'price_below',
  'change_pct_above',
  'change_pct_below',
];
