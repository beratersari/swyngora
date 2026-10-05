import type { VolumeProfileBin, VolumeProfileReport, VolumeProfileVenue } from '@/libs/api';

const ROW_LIMIT = 24;

function num(value?: string): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function profileVenue(report?: VolumeProfileReport | null): VolumeProfileVenue | undefined {
  if (!report) return undefined;
  if (report.definite !== false && report.combined?.bins && report.combined.bins.length > 0) {
    return report.combined;
  }
  return report.venues?.find((v) => (v.bins?.length ?? 0) > 0 && !v.error) ?? report.venues?.[0];
}

/** Venues whose stored prints start after the window open. */
export function profileMissing(report?: VolumeProfileReport | null): string[] {
  return report?.definite === false ? (report.missing ?? []) : [];
}

function activity(row: VolumeProfileBin): number {
  return num(row.oiOpened) + num(row.oiClosed) + num(row.longLiq) + num(row.shortLiq);
}

/** Price rows where volume, open interest, or liquidations actually sit, highest price first. */
export function profileLadderRows(bins?: VolumeProfileBin[]): VolumeProfileBin[] {
  const rows = (bins ?? []).filter((b) => num(b.volume) > 0 || activity(b) > 0);
  const hot = rows.filter((b) => b.inValueArea || b.isPoc || activity(b) > 0);
  const chosen = (hot.length > 0 ? hot : rows).slice();
  chosen.sort((a, b) => num(b.price) - num(a.price));
  if (chosen.length <= ROW_LIMIT) return chosen;
  const poc = chosen.find((b) => b.isPoc) ?? chosen[Math.floor(chosen.length / 2)];
  const pocPx = num(poc?.price);
  return chosen
    .slice()
    .sort((a, b) => Math.abs(num(a.price) - pocPx) - Math.abs(num(b.price) - pocPx))
    .slice(0, ROW_LIMIT)
    .sort((a, b) => num(b.price) - num(a.price));
}

export function splitPct(left?: string, right?: string, peak?: number): { left: number; right: number } {
  const a = num(left);
  const b = num(right);
  const scale = peak && peak > 0 ? peak : a + b;
  if (scale <= 0) return { left: 0, right: 0 };
  return { left: (a / scale) * 100, right: (b / scale) * 100 };
}

export function columnPeak(rows: VolumeProfileBin[], left: keyof VolumeProfileBin, right: keyof VolumeProfileBin): number {
  let peak = 0;
  for (const row of rows) {
    const sum = num(row[left] as string | undefined) + num(row[right] as string | undefined);
    if (sum > peak) peak = sum;
  }
  return peak;
}
