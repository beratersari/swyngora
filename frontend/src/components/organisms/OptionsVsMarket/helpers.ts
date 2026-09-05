import type { MarketOptionsFinding } from '@/libs/api';

export function vsMarketFindings(
  findings: MarketOptionsFinding[] | undefined,
): MarketOptionsFinding[] {
  if (!findings?.length) return [];
  return findings.filter((row) => Boolean(row.title));
}
