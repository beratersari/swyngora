export { baseApi } from './baseApi';
export { store } from './store';
export type { RootState, AppDispatch } from './store';
export { useAppDispatch, useAppSelector } from './hooks';
export { useGetHealthQuery } from './endpoints/healthApi';
export type { HealthResponse } from './endpoints/healthApi';
export {
  marketApi,
  useListExchangesQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
  useListIntervalsQuery,
  useGetCandlesQuery,
  useLazyGetCandlesQuery,
  useGetTicker24hQuery,
  useGetSupplyQuery,
  useGetIndicatorsQuery,
  usePostIndicatorsBatchQuery,
  useLazyPostIndicatorsBatchQuery,
  compactParams,
} from './endpoints/marketApi';
export type {
  SpotMarket,
  SpotListResponse,
  SpotListQuery,
  SpotSortField,
  SpotSortOrder,
  MarketExchange,
  ExchangesResponse,
  ProductTagsResponse,
  CandlesResponse,
  Candle,
  Ticker24h,
  Supply,
  CandlesQuery,
  Ticker24hQuery,
  SupplyQuery,
  IntervalsQuery,
  IndicatorsQuery,
  IntervalsResponse,
  IndicatorsResponse,
  IndicatorsBatchArg,
  IndicatorsBatchItem,
  IndicatorsBatchResponse,
} from './endpoints/marketApi';
export {
  watchlistApi,
  useGetWatchlistQuery,
  useLazyGetWatchlistQuery,
  useAddWatchlistItemMutation,
  useRemoveWatchlistItemMutation,
  useReplaceWatchlistMutation,
} from './endpoints/watchlistApi';
export type {
  Watchlist,
  WatchlistItem,
  AddWatchlistItemArg,
  RemoveWatchlistItemArg,
  ReplaceWatchlistArg,
} from './endpoints/watchlistApi';
export { rtkErrorMessage } from './rtkErrorMessage';

export {
  pumpApi,
  useScanPumpEventsQuery,
  useGetPumpEventsQuery,
} from './endpoints/pumpApi';
export type {
  PumpEvent,
  PumpEventsResponse,
  PumpScanHit,
  PumpScanResponse,
  ScanPumpEventsQuery,
  GetPumpEventsQuery,
  PumpDirection,
  PumpMode,
} from './endpoints/pumpApi';

export { aiApi, usePostAiChatMutation, buildAiChatBody } from './endpoints/aiApi';
export type {
  AiChatRequest,
  AiChatResponse,
  PostAiChatArg,
} from './endpoints/aiApi';
