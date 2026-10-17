import type { OrderPreview } from '@/libs/api';
import { formatPrice } from '@/libs/utils/formatPrice';
import { PREVIEW_WARNING_IDS, type PreviewWarningId } from './constants';
import type { PreviewRow } from './OrderPreview.types';

/** Whole minutes and leftover seconds for a spread gap. */
export function spreadEvery(seconds: number | null | undefined): { min: number; sec: number } {
  const total = Math.max(0, Math.round(seconds ?? 0));
  return { min: Math.floor(total / 60), sec: total % 60 };
}

/** One decimal, without locale grouping, so 40 and 12.5 stay stable in tests. */
export function formatPreviewPct(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '—';
  const rounded = Math.round(value * 10) / 10;
  return `${rounded}%`;
}

function push(rows: PreviewRow[], id: string, labelKey: string, value: string) {
  rows.push({ id, labelKey, value });
}

function pushPrice(
  rows: PreviewRow[],
  id: string,
  labelKey: string,
  value: number | null | undefined,
  when = true,
) {
  if (!when || value == null || !Number.isFinite(value)) return;
  push(rows, id, labelKey, formatPrice(value));
}

/** Cost lines for the ticket. Amounts use the same formatter as the rest of the desk. */
export function previewRows(preview: OrderPreview): PreviewRow[] {
  const rows: PreviewRow[] = [];
  const margin = preview.product === 'margin';
  pushPrice(rows, 'fill', 'fill', preview.fillPrice, (preview.fillPrice ?? 0) > 0);
  pushPrice(rows, 'fee', 'fee', preview.fee ?? 0);
  pushPrice(rows, 'debit', 'debit', preview.cashDebit, (preview.cashDebit ?? 0) > 0);
  pushPrice(rows, 'credit', 'credit', preview.cashCredit, (preview.cashCredit ?? 0) > 0);
  pushPrice(
    rows,
    'reservedCash',
    'reservedCash',
    preview.reservedCash,
    (preview.reservedCash ?? 0) > 0,
  );
  pushPrice(
    rows,
    'reservedQty',
    'reservedQty',
    preview.reservedQuantity,
    (preview.reservedQuantity ?? 0) > 0,
  );
  if (margin) {
    pushPrice(
      rows,
      'initialMargin',
      'initialMargin',
      preview.initialMargin,
      (preview.initialMargin ?? 0) > 0,
    );
    pushPrice(
      rows,
      'liquidation',
      'liquidation',
      preview.liquidationPrice,
      (preview.liquidationPrice ?? 0) > 0,
    );
  }
  if ((preview.borrowedPrincipal ?? 0) > 0) {
    const asset = preview.borrowedAsset ? ` ${preview.borrowedAsset}` : '';
    push(rows, 'borrowed', 'borrowed', `${formatPrice(preview.borrowedPrincipal)}${asset}`);
  }
  if (preview.sizing) {
    pushPrice(rows, 'riskBudget', 'riskBudget', preview.sizing.riskBudget);
    pushPrice(rows, 'lossAtStop', 'lossAtStop', preview.sizing.estimatedLoss);
  }
  if (preview.estimatedRealizedPnl != null && Number.isFinite(preview.estimatedRealizedPnl)) {
    push(rows, 'pnl', 'pnl', formatPrice(preview.estimatedRealizedPnl));
  }
  pushPrice(rows, 'cashAfter', 'cashAfter', preview.cashAfter);
  pushPrice(rows, 'availableAfter', 'availableAfter', preview.availableAfter);
  return rows;
}

/** Flags the ticket translates. Server reason strings stay on the assistant response. */
export function previewWarnings(preview: OrderPreview): PreviewWarningId[] {
  const risk = preview.risk;
  const hit: Record<PreviewWarningId, boolean> = {
    blocked: Boolean(risk?.blocked),
    dailyLoss: Boolean(risk?.dailyLossLimitHit),
    weight: Boolean(risk?.weightLimitHit),
    exposure: Boolean(risk?.exposureLimitHit),
    cash: Boolean(preview.insufficientCash),
    quantity: Boolean(preview.insufficientQuantity),
    borrow: Boolean(preview.borrowLimitHit),
    stopBeyondLiquidation: Boolean(preview.sizing?.stopBeyondLiquidation),
    book: Boolean(preview.bookExhausted),
    slippage: Boolean(preview.slippageLimited),
  };
  return PREVIEW_WARNING_IDS.filter((id) => hit[id]);
}
