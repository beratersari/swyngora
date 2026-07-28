/**
 * Open-cap "∞ / n/a" only when a supply snapshot exists and max is null/undefined.
 * Missing supply (still loading / failed) must not look like unlimited supply.
 */
export function formatMaxSupply(
  supply: { maxSupply?: number | null } | null | undefined,
  openLabel: string,
  formatNum: (v: number | null | undefined) => string,
): string {
  if (!supply) return '—';
  if (supply.maxSupply === null || supply.maxSupply === undefined) return openLabel;
  return formatNum(supply.maxSupply);
}
