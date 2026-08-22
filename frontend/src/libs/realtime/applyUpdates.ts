import { marketApi } from '@/libs/api/endpoints/marketApi';
import { portfolioApi } from '@/libs/api/endpoints/portfolioApi';
import type { AppDispatch, RootState } from '@/libs/api/store';
import type { RealtimePortfolioEvent, RealtimePriceTick } from './realtime.types';

type SpotListItem = {
  exchange?: string;
  symbol?: string;
  lastPrice?: string;
  priceChange?: string;
  priceChangePercent?: string;
  openPrice?: string;
  highPrice?: string;
  lowPrice?: string;
  volume?: string;
  quoteVolume?: string;
  marketCapCirculating?: string | number | null;
  marketCapTotal?: string | number | null;
  marketCapMax?: string | number | null;
  tradeCount?: number | string | null;
};

function scaleCap(value: string | number | null | undefined, ratio: number): string | number | null | undefined {
  if (value == null || value === '') return value;
  const n = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(n)) return value;
  const next = n * ratio;
  return typeof value === 'number' ? next : String(next);
}

/** Patch one list row from a tick. Caller must already scope the list by exchange. */
export function patchSpotItem(item: SpotListItem, tick: RealtimePriceTick): void {
  if ((item.symbol ?? '').toUpperCase() !== (tick.symbol ?? '').toUpperCase()) return;

  const oldPrice = Number(item.lastPrice);
  const newPrice = tick.lastPrice != null ? Number(tick.lastPrice) : NaN;
  if (
    tick.lastPrice != null &&
    Number.isFinite(oldPrice) &&
    oldPrice > 0 &&
    Number.isFinite(newPrice) &&
    newPrice > 0
  ) {
    const ratio = newPrice / oldPrice;
    if (ratio !== 1 && Number.isFinite(ratio)) {
      // Keep default mcap columns coherent with live last while WS is connected.
      item.marketCapCirculating = scaleCap(
        item.marketCapCirculating,
        ratio,
      ) as SpotListItem['marketCapCirculating'];
      item.marketCapTotal = scaleCap(item.marketCapTotal, ratio) as SpotListItem['marketCapTotal'];
      item.marketCapMax = scaleCap(item.marketCapMax, ratio) as SpotListItem['marketCapMax'];
    }
  }

  if (tick.lastPrice != null) item.lastPrice = tick.lastPrice;
  if (tick.priceChange != null) item.priceChange = tick.priceChange;
  if (tick.priceChangePercent != null) item.priceChangePercent = tick.priceChangePercent;
  if (tick.openPrice != null) item.openPrice = tick.openPrice;
  if (tick.highPrice != null) item.highPrice = tick.highPrice;
  if (tick.lowPrice != null) item.lowPrice = tick.lowPrice;
  if (tick.volume != null) item.volume = tick.volume;
  if (tick.quoteVolume != null) item.quoteVolume = tick.quoteVolume;
}

function listExchangeFromArgs(args: unknown): string {
  if (args && typeof args === 'object' && 'exchange' in args) {
    const ex = (args as { exchange?: unknown }).exchange;
    if (typeof ex === 'string') return ex.toLowerCase();
  }
  return '';
}

export function applyPriceTick(dispatch: AppDispatch, getState: () => RootState, tick: RealtimePriceTick): void {
  const exchange = (tick.exchange || 'binance') as
    | 'binance'
    | 'coinbase'
    | 'bybit'
    | 'nasdaq'
    | 'bist';
  const symbol = tick.symbol;
  if (!symbol) return;
  const tickEx = exchange.toLowerCase();

  dispatch(
    marketApi.util.updateQueryData('getTicker24h', { exchange, symbol }, (draft) => {
      if (!draft) return;
      if (tick.lastPrice != null) draft.lastPrice = tick.lastPrice;
      if (tick.priceChange != null) draft.priceChange = tick.priceChange;
      if (tick.priceChangePercent != null) draft.priceChangePercent = tick.priceChangePercent;
      if (tick.openPrice != null) draft.openPrice = tick.openPrice;
      if (tick.highPrice != null) draft.highPrice = tick.highPrice;
      if (tick.lowPrice != null) draft.lowPrice = tick.lowPrice;
      if (tick.volume != null) draft.volume = tick.volume;
      if (tick.quoteVolume != null) draft.quoteVolume = tick.quoteVolume;
      if (tick.tradeCount != null) draft.tradeCount = tick.tradeCount;
      if (tick.openTime) draft.openTime = tick.openTime;
      if (tick.closeTime) draft.closeTime = tick.closeTime;
      if (tick.halted != null) draft.halted = tick.halted;
      draft.exchange = exchange;
      draft.symbol = symbol;
    }),
  );

  const queries = getState().api.queries;
  for (const q of Object.values(queries)) {
    if (!q || q.endpointName !== 'listSpotMarkets') continue;
    const listEx = listExchangeFromArgs(q.originalArgs);
    // SpotMarket rows have no exchange field — scope by the list query arg only.
    if (!listEx || listEx !== tickEx) continue;
    dispatch(
      marketApi.util.updateQueryData('listSpotMarkets', q.originalArgs as never, (draft) => {
        if (!draft?.items) return;
        for (const item of draft.items) {
          patchSpotItem(item as SpotListItem, tick);
        }
      }),
    );
  }
}

export function applyPortfolioEvent(
  dispatch: AppDispatch,
  ev: RealtimePortfolioEvent,
  getState?: () => RootState,
): void {
  const view = ev.portfolio;
  if (!view) {
    dispatch(portfolioApi.util.invalidateTags(['Portfolio']));
    return;
  }
  const id = ev.portfolioId || view.id;
  dispatch(
    portfolioApi.util.updateQueryData('getPortfolio', id ? { portfolioId: id } : undefined, () => view),
  );
  dispatch(
    portfolioApi.util.updateQueryData('getPortfolio', undefined, (draft) => {
      if (!draft) return view;
      if (id && draft.id && draft.id !== id) return draft;
      return view;
    }),
  );

  const state = getState?.();
  if (state) {
    for (const q of Object.values(state.api.queries)) {
      if (!q) continue;
      const args = q.originalArgs as { portfolioId?: string } | void;
      const pid = args && typeof args === 'object' ? args.portfolioId : undefined;
      if (id && pid && pid !== id) continue;
      if (q.endpointName === 'listPortfolioOrders' && ev.orders) {
        dispatch(
          portfolioApi.util.updateQueryData('listPortfolioOrders', q.originalArgs as never, (draft) => {
            if (!draft) return;
            draft.orders = ev.orders as typeof draft.orders;
            draft.count = ev.orders?.length;
          }),
        );
      }
      if (q.endpointName === 'listMarginPositions') {
        const rows = view.marginPositions;
        if (rows) {
          dispatch(
            portfolioApi.util.updateQueryData(
              'listMarginPositions',
              q.originalArgs as never,
              (draft) => {
                if (!draft) return;
                (draft as { positions?: unknown }).positions = rows;
              },
            ),
          );
        }
      }
    }
  }

  // Refetch orders/trades/margin so polling caches cannot stay on a stale snapshot.
  dispatch(portfolioApi.util.invalidateTags(['Portfolio']));
}
