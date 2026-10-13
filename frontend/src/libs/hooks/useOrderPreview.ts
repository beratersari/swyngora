import { usePreviewPortfolioOrderQuery, type PreviewPortfolioOrderArg } from '@/libs/api';
import {
  ORDER_PREVIEW_DEBOUNCE_MS,
  ORDER_PREVIEW_POLL_MS,
  previewDraftReady,
  toPreviewArg,
  type OrderPreviewDraft,
} from './orderPreview';
import { useDebouncedValue } from './useDebouncedValue';

const idleArg: PreviewPortfolioOrderArg = {
  symbol: '_',
  quantity: 0,
  product: 'spot',
  type: 'market',
};

export type UseOrderPreviewOptions = {
  portfolioId?: string;
  /** False while the page has no selected book. */
  enabled?: boolean;
  /** False while the tab is hidden. Polling pauses; the last preview stays. */
  visible?: boolean;
};

/**
 * Debounced read of what the current ticket would cost. Nothing is placed.
 * The page owns this hook. Tickets only report the draft and render the result.
 */
export function useOrderPreview(draft: OrderPreviewDraft | null, options: UseOrderPreviewOptions = {}) {
  const enabled = options.enabled !== false;
  const visible = options.visible !== false;
  const debounced = useDebouncedValue(draft, ORDER_PREVIEW_DEBOUNCE_MS);
  const ready = enabled && previewDraftReady(debounced);
  const arg = ready ? toPreviewArg(debounced, options.portfolioId) : idleArg;
  const query = usePreviewPortfolioOrderQuery(arg, {
    skip: !ready,
    pollingInterval: ready && visible ? ORDER_PREVIEW_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const preview = ready ? query.currentData : undefined;
  return {
    preview,
    loading: ready && query.isFetching && preview == null,
    failed: ready && query.isError && preview == null,
  };
}
