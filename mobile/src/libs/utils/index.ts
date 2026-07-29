export { formatPrice } from './formatPrice';
export {
  formatChangePercent,
  changeTone,
  formatCompactUsd,
  formatTradeCount,
  formatSupplyNum,
} from './formatMarket';
export {
  DEFAULT_MARKETS_FILTER,
  toSpotListQuery,
  normalizeExchange,
  isMarketExchange,
  isSpotSortField,
} from './spotQuery';
export type { MarketsFilterState } from './spotQuery';
export {
  apiCandlesToChart,
  mergeChartCandles,
  endTimeBeforeOldestCandle,
  indicatorPointsToEmaLine,
  indicatorPointsToRsi,
  sortedEmaKeys,
  resolveInterval,
  emaColor,
  EMA_LINE_COLORS,
} from './candles';
export type { ApiCandle, ChartCandle, ChartLinePoint } from './candles';

export { getOrCreateClientId, peekClientId } from './clientId';
export { watchKey, normalizePair } from './watchlistKey';
export type { WatchlistPair } from './watchlistKey';
export {
  mergeWatchlists,
  isAtMaxItems,
  readLocalWatchlist,
  serializeLocalWatchlist,
} from './watchlistMerge';
export { appStorage, createTestStorage } from './storage';
export type { KeyValueStorage } from './storage';

export {
  formatPumpReturnPct,
  pumpReturnTone,
  formatVolumeRatio,
  pumpModeLabel,
  formatPumpEventTime,
} from './formatPump';
export {
  defaultPumpScanFilters,
  defaultQuoteForExchange,
  buildScanQuery,
  buildDetailPumpQuery,
} from './pumpQuery';
export type {
  PumpScanFilterState,
  PumpDetailQueryState,
  PumpDirection,
  PumpMode,
} from './pumpQuery';
export {
  pumpEventsToChartMarkers,
  pumpEventsToMarginLines,
} from './pumpChart';
export type { ChartMarker, ChartPriceLine } from './pumpChart';
export {
  chunkSymbols,
  groupPairsByExchange,
  buildBatchIndicatorsArg,
  buildBatchIndicatorsBody,
  indexBatchItemsBySymbol,
  batchItemKey,
  formatRsi,
  rsiTone,
  rsiFieldsFromItem,
} from './batchIndicators';
export type {
  BatchIndicatorsArg,
  BatchIndicatorItem,
  BatchIndicatorsResponse,
  RsiRowFields,
} from './batchIndicators';

export {
  getOrCreateAiSessionId,
  rotateAiSessionId,
  peekAiSessionId,
  resetAiSessionCacheForTests,
} from './aiSession';
export {
  buildContextPrompt,
  createMessageId,
  createUserMessage,
  createAssistantMessage,
  createPendingAssistantMessage,
  trimMessages,
} from './aiContextPrompt';
export type {
  AiContextParams,
  ChatRole,
  ChatMessageModel,
} from './aiContextPrompt';

export {
  buildMoversSpotQuery,
  buildVolumeSpotQuery,
  buildHomePumpScanQuery,
  mapSpotToDashboardRow,
  mapSpotListToDashboardRows,
  mapFavoritesToDashboardRows,
  mapPumpHitsToTeasers,
  indexDashboardRows,
} from './homeDashboardQuery';
export type {
  DashboardMarketRow,
  DashboardPumpTeaser,
} from './homeDashboardQuery';

export {
  intersectFeaturedTags,
  filterTagsBySearch,
  formatCategoryLabel,
  buildCategorySpotParams,
} from './categoryQuery';
export type { CategorySpotParamsInput } from './categoryQuery';
