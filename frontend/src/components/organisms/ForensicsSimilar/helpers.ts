import type { ForensicsSimilarField, ForensicsSimilarHit } from '@/libs/types/forensics.types';
import { formatForensicsTime } from '@/libs/utils/forensics';
import { SIMILAR_FIELD_ORDER } from './constants';

export function orderedSimilarFields(fields?: ForensicsSimilarField[]): ForensicsSimilarField[] {
  if (!fields?.length) return [];
  const byName = new Map<string, ForensicsSimilarField>();
  for (const f of fields) {
    if (f.name) byName.set(f.name, f);
  }
  const out: ForensicsSimilarField[] = [];
  for (const name of SIMILAR_FIELD_ORDER) {
    const row = byName.get(name);
    if (row) out.push(row);
  }
  for (const f of fields) {
    if (f.name && !SIMILAR_FIELD_ORDER.includes(f.name as (typeof SIMILAR_FIELD_ORDER)[number])) {
      out.push(f);
    }
  }
  return out;
}

export function similarFieldScore(field: ForensicsSimilarField): string {
  if (!field.used) return 'n/a';
  return field.score && field.score !== '' ? field.score : '—';
}

export function similarMoveAt(hit: ForensicsSimilarHit | undefined): string | null {
  const raw = hit?.at?.trim();
  if (!raw) return null;
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return null;
  return d.toISOString();
}

const SIMILAR_ASOF_SLACK_MS = 2 * 60 * 1000;

/** A label when the comparison clock is already in the past. A live read stays quiet. */
export function similarAsOfLabel(asOf: string | undefined, nowMs: number = Date.now()): string | null {
  if (!asOf) return null;
  const ms = Date.parse(asOf);
  if (!Number.isFinite(ms) || nowMs - ms < SIMILAR_ASOF_SLACK_MS) return null;
  return formatForensicsTime(asOf);
}
