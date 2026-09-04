import type {
  ForensicsPhase,
  ForensicsPhaseId,
  ForensicsTape,
  ForensicsVenue,
} from '@/libs/types/forensics.types';

export function pickForensicsVenue(
  tape: ForensicsTape | undefined,
  venueId: string,
): ForensicsVenue | undefined {
  if (!tape) return undefined;
  if (venueId === 'combined' && tape.combined) return tape.combined;
  const match = tape.venues?.find((v) => v.exchange === venueId);
  if (match) return match;
  if (tape.combined) return tape.combined;
  return tape.venues?.[0];
}

export function forensicsVenueOptions(
  tape: ForensicsTape | undefined,
): { value: string; label: string }[] {
  const out: { value: string; label: string }[] = [];
  if (tape?.combined) out.push({ value: 'combined', label: 'combined' });
  for (const v of tape?.venues ?? []) {
    if (v.exchange) out.push({ value: v.exchange, label: v.exchange });
  }
  return out;
}

export function forensicsPhaseById(
  venue: ForensicsVenue | undefined,
  id: ForensicsPhaseId,
): ForensicsPhase | undefined {
  return venue?.phases?.find((p) => p.phase === id);
}

export function formatForensicsTime(iso: string | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toISOString().replace('T', ' ').replace(/\.\d+Z$/, ' UTC');
}
