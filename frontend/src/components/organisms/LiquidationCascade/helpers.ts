import type { CascadeGrade, CascadeSide, CascadeVenue, CascadeWindow } from './LiquidationCascade.types';
import { CASCADE_WINDOWS } from './constants';

export function parseRatio(value: number | undefined): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

export function formatRatio(value: number | undefined): string {
  const n = parseRatio(value);
  if (n <= 0) return '—';
  return `${n.toFixed(n >= 10 ? 0 : 1)}×`;
}

export function gradeTone(grade: CascadeGrade | undefined): 'quiet' | 'elevated' | 'cascade' | 'extreme' {
  if (grade === 'elevated' || grade === 'cascade' || grade === 'extreme') return grade;
  return 'quiet';
}

export function sideTone(side: CascadeSide | undefined): 'long' | 'short' | 'both' | 'none' {
  if (side === 'long' || side === 'short' || side === 'both') return side;
  return 'none';
}

export function orderedWindows(windows: CascadeWindow[] | undefined): CascadeWindow[] {
  const list = windows ?? [];
  return CASCADE_WINDOWS.map((id) => list.find((w) => w.window === id) ?? { window: id });
}

export function venueLabel(exchange: string | undefined): string {
  if (!exchange) return '';
  return exchange.charAt(0).toUpperCase() + exchange.slice(1);
}

export function hottestWindow(venue: CascadeVenue | undefined): CascadeWindow | undefined {
  if (!venue) return undefined;
  return (venue.windows ?? []).find((w) => w.window === venue.hottest);
}
