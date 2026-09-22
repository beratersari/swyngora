import { formatTapeNum } from '@/components/organisms/TapePanel/TapePanel.helpers';

export function signedTapeAmount(raw?: string | null): string {
  if (raw == null || raw === '') return '—';
  const n = Number(raw);
  if (!Number.isFinite(n)) return raw;
  const body = formatTapeNum(Math.abs(n));
  if (n > 0) return `+${body}`;
  if (n < 0) return `-${body}`;
  return body;
}

export function sideLine(netLong?: string | null, netShort?: string | null): string {
  return `Long ${signedTapeAmount(netLong)} · Short ${signedTapeAmount(netShort)}`;
}
