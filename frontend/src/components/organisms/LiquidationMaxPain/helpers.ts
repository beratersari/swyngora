import type { MaxPainPocket } from './LiquidationMaxPain.types';

export function venueLabel(exchange?: string): string {
  const v = (exchange ?? '').toLowerCase();
  if (v === 'binance') return 'Binance';
  if (v === 'bybit') return 'Bybit';
  return exchange || 'Venue';
}

export function sideTone(side?: string): 'up' | 'down' {
  return side === 'long' ? 'down' : 'up';
}

export function parseNum(raw?: string | number | null): number | null {
  if (raw == null || raw === '') return null;
  const n = typeof raw === 'number' ? raw : Number(raw);
  return Number.isFinite(n) ? n : null;
}

export function pocketLevels(primary?: MaxPainPocket, rest?: MaxPainPocket[]): MaxPainPocket[] {
  const rows = rest ?? [];
  if (!primary?.price) return rows;
  return rows.filter((row) => row.price !== primary.price);
}
