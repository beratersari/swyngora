import type { PaperOrderType } from '@/libs/api';

export type SpotOrderKind =
  | 'market'
  | 'limit'
  | 'stop_loss'
  | 'trailing_stop'
  | 'oco'
  | 'bracket';

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


