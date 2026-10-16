import type { CoinSituation, PaperOrderType } from '@/libs/api';

export type SpotOrderKind = 'market' | 'limit' | 'stop_loss' | 'trailing_stop' | 'oco' | 'bracket';

export function kindFromOrderType(type: PaperOrderType | undefined): SpotOrderKind {
  switch (type) {
    case 'limit_buy':
    case 'limit_sell':
      return 'limit';
    case 'stop_loss':
      return 'stop_loss';
    case 'trailing_stop':
      return 'trailing_stop';
    case 'oco':
      return 'oco';
    case 'bracket':
      return 'bracket';
    default:
      return 'market';
  }
}

export function toApiOrderType(kind: SpotOrderKind, side: 'buy' | 'sell'): PaperOrderType {
  if (kind === 'market') return 'market';
  if (kind === 'limit') return side === 'buy' ? 'limit_buy' : 'limit_sell';
  if (kind === 'stop_loss') return 'stop_loss';
  if (kind === 'trailing_stop') return 'trailing_stop';
  if (kind === 'oco') return 'oco';
  return 'bracket';
}

export const REASON_CLAUSE_IDS = [
  'priceCvd',
  'openInterest',
  'liquidation',
  'prints',
  'funding',
  'crowd',
  'size',
] as const;

export type ReasonClauseId = (typeof REASON_CLAUSE_IDS)[number];

export type ReasonSituation = {
  priceCvd?: { kind?: string } | null;
  openInterest?: { flow?: string } | null;
  liquidation?: { side?: string } | null;
  prints?: { side?: string } | null;
  funding?: { payer?: string } | null;
  crowd?: { lean?: string } | null;
  size?: { kind?: string } | null;
};

/** A spot buy, limit buy, or bracket stores a reason. Equities and sells do not. */
export function orderKeepsReason(
  exchange: string,
  kind: SpotOrderKind,
  side: 'buy' | 'sell',
): boolean {
  if (exchange === 'nasdaq' || exchange === 'bist') return false;
  if (kind === 'bracket') return true;
  if (kind === 'market' || kind === 'limit') return side === 'buy';
  return false;
}

export function presentReasonClauses(sit: ReasonSituation | null | undefined): ReasonClauseId[] {
  if (!sit) return [];
  return REASON_CLAUSE_IDS.filter((id) => {
    switch (id) {
      case 'priceCvd':
        return Boolean(sit.priceCvd?.kind);
      case 'openInterest':
        return Boolean(sit.openInterest?.flow);
      case 'liquidation':
        return Boolean(sit.liquidation?.side);
      case 'prints':
        return Boolean(sit.prints?.side);
      case 'funding':
        return Boolean(sit.funding?.payer);
      case 'crowd':
        return Boolean(sit.crowd?.lean);
      case 'size':
        return Boolean(sit.size?.kind);
      default:
        return false;
    }
  });
}

/** One clause, so the existing sentence helper can label a checkbox. */
export function reasonClauseReport(sit: ReasonSituation, id: ReasonClauseId): CoinSituation {
  switch (id) {
    case 'priceCvd':
      return { priceCvd: sit.priceCvd } as CoinSituation;
    case 'openInterest':
      return { openInterest: sit.openInterest } as CoinSituation;
    case 'liquidation':
      return { liquidation: sit.liquidation } as CoinSituation;
    case 'prints':
      return { prints: sit.prints } as CoinSituation;
    case 'funding':
      return { funding: sit.funding } as CoinSituation;
    case 'crowd':
      return { crowd: sit.crowd } as CoinSituation;
    case 'size':
      return { size: sit.size } as CoinSituation;
    default:
      return {};
  }
}

export function needsTrigger(kind: SpotOrderKind): boolean {
  return kind === 'limit' || kind === 'stop_loss' || kind === 'bracket';
}

export function needsTpSl(kind: SpotOrderKind): boolean {
  return kind === 'oco' || kind === 'bracket';
}

export function needsTrail(kind: SpotOrderKind): boolean {
  return kind === 'trailing_stop';
}

export function needsSide(kind: SpotOrderKind): boolean {
  return kind === 'market' || kind === 'limit';
}

export function needsTif(kind: SpotOrderKind): boolean {
  return kind === 'limit' || kind === 'stop_loss';
}

/** A spot buy or a bracket can be sized from a stop. Sells and exits cannot. */
export function canSizeByRisk(kind: SpotOrderKind, side: 'buy' | 'sell'): boolean {
  if (kind === 'bracket') return true;
  return (kind === 'market' || kind === 'limit') && side === 'buy';
}
