import type { PortfolioStressReplayMove } from '@/libs/api';

export function money(n: number | undefined, signed = false): string {
  if (n == null || !Number.isFinite(n)) return '—';
  const body = Math.abs(n).toLocaleString(undefined, { maximumFractionDigits: 2 });
  if (!signed) return n < 0 ? `-${body}` : body;
  if (n > 0) return `+${body}`;
  if (n < 0) return `-${body}`;
  return body;
}

/** Percent change of the book. Blank when current equity is not a positive base. */
export function equityChangePct(equity?: number, delta?: number): number | undefined {
  if (equity == null || delta == null || !(equity > 0) || !Number.isFinite(equity) || !Number.isFinite(delta)) {
    return undefined;
  }
  const pct = (delta / equity) * 100;
  if (!Number.isFinite(pct)) return undefined;
  return pct;
}

/** Signed money change, with the percent of current equity beside it. */
export function portfolioChange(equity?: number, delta?: number): string {
  const amount = money(delta, true);
  const pct = equityChangePct(equity, delta);
  if (pct == null) return amount;
  return `${amount}, ${formatMovePct(pct)}`;
}

export function formatMovePct(n: number | undefined): string {
  if (n == null || !Number.isFinite(n)) return '—';
  const body = Math.abs(n).toLocaleString(undefined, { maximumFractionDigits: 2 });
  if (n > 0) return `+${body}%`;
  if (n < 0) return `−${body}%`;
  return `${body}%`;
}

/** UTC calendar day, offset from today's UTC date. */
export function utcDay(offsetDays: number, now = new Date()): string {
  const d = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
  d.setUTCDate(d.getUTCDate() + offsetDays);
  const y = d.getUTCFullYear();
  const m = String(d.getUTCMonth() + 1).padStart(2, '0');
  const day = String(d.getUTCDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export type ReplayPreset = 'yesterday' | '7' | '30';

export function replayPreset(kind: ReplayPreset, now = new Date()): { from: string; to: string } {
  if (kind === 'yesterday') {
    const day = utcDay(-1, now);
    return { from: day, to: day };
  }
  const n = kind === '7' ? 7 : 30;
  return { from: utcDay(-n, now), to: utcDay(-1, now) };
}

export function replayMoveLine(move: PortfolioStressReplayMove): string {
  const pct = formatMovePct(move.movePct);
  const asset = move.asset ?? '';
  const venues = move.venues ?? [];
  if (venues.length > 1) {
    const detail = venues
      .map((v) => `${v.exchange ?? v.symbol ?? ''} ${formatMovePct(v.movePct)}`.trim())
      .join(', ');
    return `${asset} ${pct} · ${detail}`.trim();
  }
  if (venues.length === 1) {
    const v = venues[0];
    return `${asset} ${pct} · ${money(v.start)} → ${money(v.end)}`.trim();
  }
  return `${asset} ${pct}`.trim();
}
