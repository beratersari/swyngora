import type { ForensicsSimilarField } from '@/libs/types/forensics.types';
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
