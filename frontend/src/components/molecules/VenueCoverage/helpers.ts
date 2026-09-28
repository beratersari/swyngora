import type { VenueRead } from './VenueCoverage.types';

export function coverageAge(ageSec: number | undefined): string {
  const sec = ageSec ?? 0;
  if (sec <= 0) return '';
  if (sec < 90) return `${sec}s`;
  if (sec < 90 * 60) return `${Math.floor(sec / 60)}m`;
  return `${Math.floor(sec / 3600)}h`;
}

export function venueStatusKey(status: string | undefined): 'fresh' | 'stale' | 'missing' {
  if (status === 'fresh' || status === 'stale') return status;
  return 'missing';
}

export function venueRows(venues: VenueRead[] | undefined): VenueRead[] {
  return venues ?? [];
}
