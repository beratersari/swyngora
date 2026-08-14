import type { PaperOrderType } from '@/libs/api';
import type { PaperTradeFormValues } from './PaperTradeForm.types';

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

export function validateTradeForm(values: PaperTradeFormValues): string | null {
  if (!values.symbol?.trim()) return 'symbol';
  if (!Number.isFinite(values.quantity) || values.quantity <= 0) return 'quantity';
  const kind = kindFromOrderType(values.orderType);
  if (needsTrigger(kind) && !(values.triggerPrice != null && values.triggerPrice > 0)) {
    return 'triggerPrice';
  }
  if (needsTpSl(kind)) {
    if (!(values.takeProfitPrice != null && values.takeProfitPrice > 0)) return 'takeProfitPrice';
    if (!(values.stopLossPrice != null && values.stopLossPrice > 0)) return 'stopLossPrice';
  }
  if (needsTrail(kind)) {
    if (!(values.trailValue != null && values.trailValue > 0)) return 'trailValue';
  }
  return null;
}
