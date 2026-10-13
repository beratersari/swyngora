import type { MarketExchange, MarginMode, OrderPreview } from '@/libs/api';
import type { MarginOrderPreviewDraft } from '@/libs/hooks/orderPreview';

export type PaperMarginFormValues = {
  exchange: MarketExchange;
  symbol: string;
  side: 'long' | 'short';
  orderType: 'market' | 'limit';
  quantity: number;
  leverage: number;
  limitPrice?: number;
  stopLoss?: number;
  takeProfit?: number;
};

export type PaperMarginFormProps = {
  marginMode?: MarginMode | string;
  modeLoading?: boolean;
  isSubmitting?: boolean;
  /** When true, open is blocked (e.g. 2+ books and none selected). */
  disabled?: boolean;
  submitError?: unknown;
  /** Latest cost and risk for this ticket. The page owns the request. */
  preview?: OrderPreview | null;
  previewLoading?: boolean;
  previewFailed?: boolean;
  /** Fires when the draft changes so the page can price it. */
  onDraftChange?: (draft: MarginOrderPreviewDraft) => void;
  onModeChange?: (mode: MarginMode) => Promise<void>;
  onSubmit: (values: PaperMarginFormValues) => Promise<void>;
};
