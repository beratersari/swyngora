import { marketApi } from '@/libs/api/endpoints/marketApi';
import { portfolioApi } from '@/libs/api/endpoints/portfolioApi';
import type { AppDispatch, RootState } from '@/libs/api/store';
import type { RealtimePortfolioEvent, RealtimePriceTick } from './realtime.types';

function patchSpotItem(
  item: {
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
  },
  tick: RealtimePriceTick,
): void {
  const itemEx = (item.exchange ?? '').toLowerCase();
  const tickEx = (tick.exchange ?? '').toLowerCase();
  if (itemEx && tickEx && itemEx !== tickEx) return;
  if ((item.symbol ?? '').toUpperCase() !== (tick.symbol ?? '').toUpperCase()) return;
  if (tick.lastPrice != null) item.lastPrice = tick.lastPrice;
  if (tick.priceChange != null) item.priceChange = tick.priceChange;
  if (tick.priceChangePercent != null) item.priceChangePercent = tick.priceChangePercent;
  if (tick.openPrice != null) item.openPrice = tick.openPrice;
  if (tick.highPrice != null) item.highPrice = tick.highPrice;
  if (tick.lowPrice != null) item.lowPrice = tick.lowPrice;
  if (tick.volume != null) item.volume = tick.volume;
  if (tick.quoteVolume != null) item.quoteVolume = tick.quoteVolume;
}

export function applyPriceTick(dispatch: AppDispatch, getState: () => RootState, tick: RealtimePriceTick): void {
  const exchange = (tick.exchange || 'binance') as 'binance' | 'coinbase' | 'bybit';
  const symbol = tick.symbol;
  if (!symbol) return;

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
      draft.exchange = exchange;
      draft.symbol = symbol;
    }),
  );

  const queries = getState().api.queries;
  for (const q of Object.values(queries)) {
    if (!q || q.endpointName !== 'listSpotMarkets') continue;
    dispatch(
      marketApi.util.updateQueryData('listSpotMarkets', q.originalArgs as never, (draft) => {
        if (!draft?.items) return;
        for (const item of draft.items) {
          patchSpotItem(item, tick);
        }
      }),
    );
  }
}

export function applyPortfolioEvent(dispatch: AppDispatch, ev: RealtimePortfolioEvent): void {
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
}
