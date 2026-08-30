import type { LiquidationFeedHealthProps } from './LiquidationFeedHealth.types';

type Feed = NonNullable<LiquidationFeedHealthProps['feed']>;
type Venue = NonNullable<Feed['venues']>[number];

export function formatClock(raw?: string): string {
  if (!raw) return '—';
  const t = Date.parse(raw);
  if (!Number.isFinite(t)) return '—';
  return new Date(t).toISOString().slice(11, 19) + 'Z';
}

export function gapHours(venue?: Venue): number {
  if (!venue?.gaps?.length) return 0;
  return venue.gaps.reduce((sum, g) => sum + (g.seconds ?? 0), 0) / 3600;
}

export function venuesOf(feed?: Feed | null): Venue[] {
  return feed?.venues ?? [];
}