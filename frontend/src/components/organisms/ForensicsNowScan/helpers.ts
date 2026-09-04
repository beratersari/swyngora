import type { ForensicsScanField } from '@/libs/types/forensics.types';
import { SCAN_FIELD_ORDER } from './constants';

export function setupRadarAlertHref(symbol: string, exchange?: string): string {
  const q = new URLSearchParams({ kind: 'setup_radar', symbol: symbol || 'ALL' });
  if (exchange) q.set('exchange', exchange);
  return `/alerts?${q.toString()}`;
}

export function orderedScanFields(fields?: ForensicsScanField[]): ForensicsScanField[] {
  if (!fields?.length) return [];
  const byName = new Map<string, ForensicsScanField>();
  for (const f of fields) {
    if (f.name) byName.set(f.name, f);
  }
  const out: ForensicsScanField[] = [];
  for (const name of SCAN_FIELD_ORDER) {
    const row = byName.get(name);
    if (row) out.push(row);
  }
  for (const f of fields) {
    if (f.name && !SCAN_FIELD_ORDER.includes(f.name as (typeof SCAN_FIELD_ORDER)[number])) {
      out.push(f);
    }
  }
  return out;
}

export function scanFieldScore(field: ForensicsScanField): string {
  if (!field.used) return 'n/a';
  return field.score && field.score !== '' ? field.score : '—';
}
