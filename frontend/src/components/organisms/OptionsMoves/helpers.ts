import type { MarketOptionsMove } from '@/libs/api';

export function unusualMoves(findings: MarketOptionsMove[] | undefined): MarketOptionsMove[] {
  if (!findings?.length) return [];
  return findings.filter((row) => Boolean(row.title));
}
