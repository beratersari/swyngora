import type { CoinSituation } from '@/libs/api';
import { formatCompactUsd, formatDateTime } from '@/libs/utils/formatMarket';

type Translate = (key: string, options?: Record<string, unknown>) => string;

export type SituationLineOptions = {
  since?: boolean;
  locale?: string;
};

function readableSince(since: string | undefined): string | undefined {
  if (!since) return undefined;
  const parsed = new Date(since);
  // A zero Go time serializes as year 1 and is not a clock.
  if (Number.isNaN(parsed.getTime()) || parsed.getUTCFullYear() < 2) return undefined;
  return since;
}

function withHeld(
  sentence: string,
  since: string | undefined,
  options: SituationLineOptions | undefined,
  t: Translate,
): string {
  const clock = readableSince(since);
  if (!options?.since || !clock) return sentence;
  const time = formatDateTime(clock, options.locale);
  if (!time || time === '—') return sentence;
  return `${sentence} ${t('detail:situation.heldSince', { time })}`;
}

export type SituationClauseChange = {
  clause?: string;
  status?: string;
  entry?: string;
  live?: string;
  missing?: string[];
};

export type SituationComparison = {
  same?: boolean;
  clauses?: SituationClauseChange[];
};

export function situationVenueList(ids: string[] | undefined, t: Translate): string {
  const names = (ids ?? [])
    .map((id) => id.trim())
    .filter(Boolean)
    .map((id) => t(`detail:situation.venues.${id}`, { defaultValue: id }));
  if (names.length <= 1) return names[0] ?? '';
  if (names.length === 2) {
    return t('detail:situation.venues.two', { a: names[0], b: names[1] });
  }
  return t('detail:situation.venues.many', {
    list: names.slice(0, -1).join(', '),
    last: names[names.length - 1],
  });
}

/** One sentence per clause. A missing venue keeps that sentence blank of a number and names who is absent. */
export function situationLines(
  report: CoinSituation | null | undefined,
  t: Translate,
  options?: SituationLineOptions,
): string[] {
  if (!report) return [];
  const lines: string[] = [];
  if (report.priceCvd?.kind) {
    lines.push(
      withHeld(
        t(`detail:situation.priceCvd.${report.priceCvd.kind}`, { defaultValue: report.priceCvd.kind }),
        report.priceCvd.since,
        options,
        t,
      ),
    );
  } else if ((report.priceCvdMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.priceCvd.missing', { venues: situationVenueList(report.priceCvdMissing, t) }),
    );
  }
  if (report.openInterest?.flow) {
    lines.push(
      withHeld(
        t(`detail:situation.openInterest.${report.openInterest.flow}`, {
          defaultValue: report.openInterest.flow,
        }),
        report.openInterest.since,
        options,
        t,
      ),
    );
  } else if ((report.openInterestMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.openInterest.missing', {
        venues: situationVenueList(report.openInterestMissing, t),
      }),
    );
  }
  if (report.liquidation?.side) {
    lines.push(
      withHeld(
        t(`detail:situation.liquidation.${report.liquidation.side}`, {
          pct: report.liquidation.distancePct ?? 0,
          defaultValue: report.liquidation.side,
        }),
        report.liquidation.since,
        options,
        t,
      ),
    );
  } else if ((report.liquidationMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.liquidation.missing', {
        venues: situationVenueList(report.liquidationMissing, t),
      }),
    );
  }
  if (report.prints?.side) {
    lines.push(
      withHeld(
        t(`detail:situation.prints.${report.prints.side}`, {
          long: formatCompactUsd(report.prints.longNotional ?? 0),
          short: formatCompactUsd(report.prints.shortNotional ?? 0),
          defaultValue: report.prints.side,
        }),
        report.prints.since,
        options,
        t,
      ),
    );
  } else if ((report.printsMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.prints.missing', {
        venues: situationVenueList(report.printsMissing, t),
      }),
    );
  }
  if (report.funding?.payer) {
    lines.push(
      withHeld(
        t(`detail:situation.funding.${report.funding.payer}`, {
          pct: report.funding.ratePct ?? 0,
          defaultValue: report.funding.payer,
        }),
        report.funding.since,
        options,
        t,
      ),
    );
  } else if ((report.fundingMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.funding.missing', {
        venues: situationVenueList(report.fundingMissing, t),
      }),
    );
  }
  if (report.crowd?.lean) {
    lines.push(
      withHeld(
        t(`detail:situation.crowd.${report.crowd.lean}`, {
          pct: report.crowd.gapPct ?? 0,
          defaultValue: report.crowd.lean,
        }),
        report.crowd.since,
        options,
        t,
      ),
    );
  } else if ((report.crowdMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.crowd.missing', {
        venues: situationVenueList(report.crowdMissing, t),
      }),
    );
  }
  if (report.size?.kind) {
    lines.push(
      withHeld(
        t(`detail:situation.size.${report.size.kind}`, { defaultValue: report.size.kind }),
        report.size.since,
        options,
        t,
      ),
    );
  } else if ((report.sizeMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.size.missing', {
        venues: situationVenueList(report.sizeMissing, t),
      }),
    );
  }
  return lines;
}

function clauseOnly(report: CoinSituation | null | undefined, clause: string): CoinSituation | undefined {
  if (!report) return undefined;
  const out: CoinSituation = { symbol: report.symbol };
  switch (clause) {
    case 'priceCvd':
      out.priceCvd = report.priceCvd;
      break;
    case 'openInterest':
      out.openInterest = report.openInterest;
      break;
    case 'liquidation':
      out.liquidation = report.liquidation;
      break;
    case 'prints':
      out.prints = report.prints;
      break;
    case 'funding':
      out.funding = report.funding;
      break;
    case 'crowd':
      out.crowd = report.crowd;
      break;
    case 'size':
      out.size = report.size;
      break;
    default:
      return undefined;
  }
  return out;
}

function clauseMissing(clause: string, missing: string[] | undefined): CoinSituation | undefined {
  if (!missing || missing.length === 0) return undefined;
  const out: CoinSituation = {};
  switch (clause) {
    case 'priceCvd':
      out.priceCvdMissing = missing;
      break;
    case 'openInterest':
      out.openInterestMissing = missing;
      break;
    case 'liquidation':
      out.liquidationMissing = missing;
      break;
    case 'prints':
      out.printsMissing = missing;
      break;
    case 'funding':
      out.fundingMissing = missing;
      break;
    case 'crowd':
      out.crowdMissing = missing;
      break;
    case 'size':
      out.sizeMissing = missing;
      break;
    default:
      return undefined;
  }
  return out;
}

const COMPARE_CLAUSES = [
  'priceCvd',
  'openInterest',
  'liquidation',
  'prints',
  'funding',
  'crowd',
  'size',
] as const;

function clauseKind(sit: CoinSituation | null | undefined, clause: string): string {
  switch (clause) {
    case 'priceCvd':
      return sit?.priceCvd?.kind ?? '';
    case 'openInterest':
      return sit?.openInterest?.flow ?? '';
    case 'liquidation':
      return sit?.liquidation?.side ?? '';
    case 'prints':
      return sit?.prints?.side ?? '';
    case 'funding':
      return sit?.funding?.payer ?? '';
    case 'crowd':
      return sit?.crowd?.lean ?? '';
    case 'size':
      return sit?.size?.kind ?? '';
    default:
      return '';
  }
}

function clauseMissingList(sit: CoinSituation | null | undefined, clause: string): string[] {
  switch (clause) {
    case 'priceCvd':
      return sit?.priceCvdMissing ?? [];
    case 'openInterest':
      return sit?.openInterestMissing ?? [];
    case 'liquidation':
      return sit?.liquidationMissing ?? [];
    case 'prints':
      return sit?.printsMissing ?? [];
    case 'funding':
      return sit?.fundingMissing ?? [];
    case 'crowd':
      return sit?.crowdMissing ?? [];
    case 'size':
      return sit?.sizeMissing ?? [];
    default:
      return [];
  }
}

/** Same rule as the Go compare: only clauses stored on the entry count. */
export function compareCoinSituation(
  entry: CoinSituation | null | undefined,
  live: CoinSituation | null | undefined,
): SituationComparison {
  if (!entry) return { same: false, clauses: [] };
  const clauses: SituationClauseChange[] = [];
  let stored = 0;
  for (const clause of COMPARE_CLAUSES) {
    const entryKind = clauseKind(entry, clause);
    if (!entryKind) continue;
    stored += 1;
    const liveKind = clauseKind(live, clause);
    if (!liveKind) {
      const missing = clauseMissingList(live, clause);
      clauses.push({
        clause,
        status: 'missing',
        entry: entryKind,
        ...(missing.length > 0 ? { missing } : {}),
      });
      continue;
    }
    if (liveKind !== entryKind) {
      clauses.push({ clause, status: 'changed', entry: entryKind, live: liveKind });
    }
  }
  return { same: stored > 0 && clauses.length === 0, clauses };
}

/** Sentences for the clauses that left. A blank later clause uses the missing-venue sentence. */
export function situationChangeLines(
  entry: CoinSituation | null | undefined,
  live: CoinSituation | null | undefined,
  comparison: SituationComparison | null | undefined,
  mode: 'live' | 'exit',
  t: Translate,
): string[] {
  if (!comparison) return [];
  if (comparison.same) {
    return [t(mode === 'exit' ? 'detail:situation.left.stillExit' : 'detail:situation.left.still')];
  }
  const clauses = comparison.clauses ?? [];
  if (clauses.length === 0) return [];
  const lines: string[] = [];
  for (const clause of clauses) {
    if (!clause.clause) continue;
    if (clause.status === 'missing') {
      const named = situationLines(clauseMissing(clause.clause, clause.missing), t);
      lines.push(named[0] ?? t('detail:situation.left.blank'));
      continue;
    }
    const was = situationLines(clauseOnly(entry, clause.clause), t)[0] ?? clause.entry ?? '';
    const now = situationLines(clauseOnly(live, clause.clause), t)[0] ?? clause.live ?? '';
    lines.push(t('detail:situation.left.wasNow', { was, now }));
  }
  return lines;
}
