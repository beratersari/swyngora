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

export function formatDurationSec(sec: number | undefined): string {
  const n = typeof sec === 'number' && Number.isFinite(sec) ? Math.max(0, Math.round(sec)) : 0;
  if (n < 60) return `${Math.max(1, n)}s`;
  const m = Math.round(n / 60);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  const rm = m % 60;
  return rm === 0 ? `${h}h` : `${h}h ${rm}m`;
}

export function formatClock(iso: string | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  const hh = String(d.getUTCHours()).padStart(2, '0');
  const mm = String(d.getUTCMinutes()).padStart(2, '0');
  return `${hh}:${mm}`;
}

export function priceMoveTone(pct: string | undefined): 'long' | 'short' | 'none' {
  if (!pct) return 'none';
  const n = Number(pct);
  if (!Number.isFinite(n) || n === 0) return 'none';
  return n < 0 ? 'long' : 'short';
}
