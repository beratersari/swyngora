import { nearestTime } from '@/components/molecules/CandleChartHost/CandleChartHost.markers';
import type { CandleChartMarker } from '@/components/molecules/CandleChartHost/CandleChartHost.types';
import { semanticColors } from '@/styles/tokens';

/** Clause order on a shared candle. Matches the situation card. */
const CLAUSE_ORDER = ['priceCvd', 'openInterest', 'prints', 'funding', 'crowd', 'size'] as const;

const CLAUSE_KINDS: Record<(typeof CLAUSE_ORDER)[number], readonly string[]> = {
  priceCvd: ['price_up_cvd_down', 'price_down_cvd_up', 'agrees'],
  openInterest: ['opened', 'closed', 'flat'],
  prints: ['long', 'short', 'even', 'quiet'],
  funding: ['long', 'short', 'none'],
  crowd: ['top_more_long', 'top_more_short', 'even'],
  size: [
    'whale_buy_retail_sell',
    'whale_sell_retail_buy',
    'both_buy',
    'both_sell',
    'whale_buy',
    'whale_sell',
    'retail_buy',
    'retail_sell',
    'flat',
  ],
};

export type SituationChangePoint = {
  clause?: string;
  at?: string;
  kind?: string;
};

function clauseIndex(clause: string): number {
  return CLAUSE_ORDER.indexOf(clause as (typeof CLAUSE_ORDER)[number]);
}

function knownKind(clause: string, kind: string): boolean {
  const index = clauseIndex(clause);
  if (index < 0) return false;
  const allowed = CLAUSE_KINDS[CLAUSE_ORDER[index]!];
  return allowed.includes(kind);
}

/**
 * One circle for the latest stored change of each situation clause.
 * Earlier changes stay on the situation read and are not drawn.
 * Several clauses on the same candle share the marker. A change farther
 * than maxDistanceSec from a candle is left off. The estimated pocket
 * is not a clause here.
 */
export function situationChangeMarkers(
  changes: readonly SituationChangePoint[] | undefined,
  candles: readonly { time: number }[],
  maxDistanceSec: number,
  label: (clause: string, kind: string) => string,
): CandleChartMarker[] {
  const times = candles.map((candle) => candle.time).filter((time) => Number.isFinite(time));
  if (!changes?.length || times.length === 0) return [];

  type Row = { clause: string; kind: string; at: number; candle: number };
  const rows: Row[] = [];
  for (const change of changes) {
    const clause = change.clause ?? '';
    const kind = change.kind ?? '';
    if (!knownKind(clause, kind) || !change.at) continue;
    const atMs = Date.parse(change.at);
    if (!Number.isFinite(atMs)) continue;
    const at = Math.floor(atMs / 1000);
    const candle = nearestTime(times, at);
    if (
      Number.isFinite(maxDistanceSec) &&
      maxDistanceSec >= 0 &&
      Math.abs(candle - at) > maxDistanceSec
    ) {
      continue;
    }
    rows.push({ clause, kind, at, candle });
  }
  if (rows.length === 0) return [];

  const latest = new Map<string, Row>();
  for (const row of rows) {
    const prev = latest.get(row.clause);
    if (!prev || row.at >= prev.at) latest.set(row.clause, row);
  }
  const byCandle = new Map<number, Map<string, Row>>();
  for (const row of latest.values()) {
    const clauses = byCandle.get(row.candle) ?? new Map<string, Row>();
    clauses.set(row.clause, row);
    byCandle.set(row.candle, clauses);
  }

  const out: CandleChartMarker[] = [];
  for (const [candle, clauses] of byCandle) {
    const parts = CLAUSE_ORDER.flatMap((clause) => {
      const row = clauses.get(clause);
      if (!row) return [];
      const text = label(row.clause, row.kind).trim();
      return text ? [text] : [];
    });
    if (parts.length === 0) continue;
    out.push({
      time: candle,
      position: 'inBar',
      color: semanticColors.chart.rsi,
      shape: 'circle',
      text: parts.join(' · '),
    });
  }
  return out.sort((a, b) => a.time - b.time);
}
