/** Truncate a chain address for display; short values stay intact. */
export function formatHolderAddress(address: string, head = 6, tail = 4): string {
  const raw = address.trim();
  if (raw.length <= head + tail + 1) return raw;
  return `${raw.slice(0, head)}…${raw.slice(-tail)}`;
}

export function formatSharePct(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '—';
  return `${value.toLocaleString(undefined, { maximumFractionDigits: 2 })}%`;
}

export function formatHolderCount(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '—';
  const abs = Math.abs(value);
  const sign = value < 0 ? '-' : '';
  if (abs >= 1e12) return `${sign}${(abs / 1e12).toFixed(2)}T`;
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(2)}B`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(2)}K`;
  return value.toLocaleString();
}

/**
 * Token quantity for a wallet. Compact for large stacks; keeps significant
 * digits for dust so 0.00041 does not print as "0".
 */
export function formatHolderBalance(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '—';
  if (value === 0) return '0';
  const abs = Math.abs(value);
  const sign = value < 0 ? '-' : '';
  if (abs >= 1e12) return `${sign}${(abs / 1e12).toFixed(2)}T`;
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(2)}B`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(2)}K`;
  if (abs >= 1) {
    return value.toLocaleString(undefined, { maximumFractionDigits: 4 });
  }
  if (abs >= 1e-6) {
    return value.toLocaleString(undefined, {
      maximumSignificantDigits: 6,
      maximumFractionDigits: 10,
    });
  }
  return value.toExponential(2);
}

/**
 * Prefer share × circulating supply when the reported CMC balance is dust-scale
 * compared with that estimate (common on high-supply tokens).
 */
export function resolveHolderBalance(
  reported: number | null | undefined,
  sharePct: number | null | undefined,
  circulatingSupply: number | null | undefined,
): number | null {
  const raw = typeof reported === 'number' && Number.isFinite(reported) ? reported : null;
  const share =
    typeof sharePct === 'number' && Number.isFinite(sharePct) ? sharePct / 100 : null;
  const circ =
    typeof circulatingSupply === 'number' && Number.isFinite(circulatingSupply) && circulatingSupply > 0
      ? circulatingSupply
      : null;
  const estimated = share != null && circ != null ? share * circ : null;
  if (estimated != null && estimated > 0) {
    if (raw == null || raw === 0) return estimated;
    if (Math.abs(estimated) / Math.max(Math.abs(raw), Number.EPSILON) >= 100) {
      return estimated;
    }
  }
  return raw;
}

export function holderUsdValue(
  sharePct: number | null | undefined,
  circulatingSupply: number | null | undefined,
  priceUsd: number | null | undefined,
): number | null {
  const share =
    typeof sharePct === 'number' && Number.isFinite(sharePct) ? sharePct / 100 : null;
  const circ =
    typeof circulatingSupply === 'number' && Number.isFinite(circulatingSupply) ? circulatingSupply : null;
  const px = typeof priceUsd === 'number' && Number.isFinite(priceUsd) ? priceUsd : null;
  if (share == null || circ == null || px == null || circ <= 0 || px <= 0) return null;
  return share * circ * px;
}
