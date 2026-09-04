import type { ForensicsRadarAfter, ForensicsRadarCase } from '@/libs/types/forensics.types';

export function radarAfterCell(row: ForensicsRadarCase, horizon: string): string {
  const a = row.after?.find((x) => x.horizon === horizon);
  if (!a) return '—';
  if (!a.filledAt) return '…';
  const bits = [a.returnPct || '0'];
  if (a.direction === 'up') bits.push('↑');
  if (a.direction === 'down') bits.push('↓');
  if (a.useful === true) bits.push('✓');
  if (a.useful === false) bits.push('✗');
  return bits.join(' ');
}

export function radarHorizonLine(horizon: string, usefulPct?: string, sample?: number): string {
  if (!sample) return `${horizon}: —`;
  if (!usefulPct) return `${horizon}: ${sample} filled`;
  return `${horizon}: ${usefulPct}% useful (n=${sample})`;
}

export function afterOf(row: ForensicsRadarCase, horizon: string): ForensicsRadarAfter | undefined {
  return row.after?.find((x) => x.horizon === horizon);
}
