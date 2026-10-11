import type { CoinSituation } from '@/libs/api';
import { formatCompactUsd } from '@/libs/utils/formatMarket';

type Translate = (key: string, options?: Record<string, unknown>) => string;

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
export function situationLines(report: CoinSituation | null | undefined, t: Translate): string[] {
  if (!report) return [];
  const lines: string[] = [];
  if (report.priceCvd?.kind) {
    lines.push(t(`detail:situation.priceCvd.${report.priceCvd.kind}`, { defaultValue: report.priceCvd.kind }));
  } else if ((report.priceCvdMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.priceCvd.missing', { venues: situationVenueList(report.priceCvdMissing, t) }),
    );
  }
  if (report.openInterest?.flow) {
    lines.push(
      t(`detail:situation.openInterest.${report.openInterest.flow}`, {
        defaultValue: report.openInterest.flow,
      }),
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
      t(`detail:situation.liquidation.${report.liquidation.side}`, {
        pct: report.liquidation.distancePct ?? 0,
        defaultValue: report.liquidation.side,
      }),
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
      t(`detail:situation.prints.${report.prints.side}`, {
        long: formatCompactUsd(report.prints.longNotional ?? 0),
        short: formatCompactUsd(report.prints.shortNotional ?? 0),
        defaultValue: report.prints.side,
      }),
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
      t(`detail:situation.funding.${report.funding.payer}`, {
        pct: report.funding.ratePct ?? 0,
        defaultValue: report.funding.payer,
      }),
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
      t(`detail:situation.crowd.${report.crowd.lean}`, {
        pct: report.crowd.gapPct ?? 0,
        defaultValue: report.crowd.lean,
      }),
    );
  } else if ((report.crowdMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.crowd.missing', {
        venues: situationVenueList(report.crowdMissing, t),
      }),
    );
  }
  if (report.size?.kind) {
    lines.push(t(`detail:situation.size.${report.size.kind}`, { defaultValue: report.size.kind }));
  } else if ((report.sizeMissing ?? []).length > 0) {
    lines.push(
      t('detail:situation.size.missing', {
        venues: situationVenueList(report.sizeMissing, t),
      }),
    );
  }
  return lines;
}
