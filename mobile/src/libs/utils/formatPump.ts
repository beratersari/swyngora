import { changeTone } from './formatMarket';

export function formatPumpReturnPct(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return '—';
  }
  const sign = value > 0 ? '+' : '';
  return `${sign}${value.toFixed(2)}%`;
}

export function pumpReturnTone(
  value: number | null | undefined,
): 'success' | 'error' | 'secondary' {
  return changeTone(value);
}

export function formatVolumeRatio(ratio: number | null | undefined): string {
  if (ratio === null || ratio === undefined || !Number.isFinite(ratio) || ratio <= 0) {
    return '';
  }
  return `vol ×${ratio.toFixed(1)}`;
}

export function pumpModeLabel(mode: string | undefined): string {
  switch (mode) {
    case 'close_return':
      return 'Close return';
    case 'candle_body':
      return 'Candle body';
    case 'high_from_low':
      return 'High from low';
    default:
      return mode || '—';
  }
}

export function formatPumpEventTime(iso: string | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}
