import type { components, operations } from '../generated/schema';

export type SpotMarket = components['schemas']['SpotMarket'];
export type SpotListResponse = components['schemas']['SpotListResponse'];
export type SpotListQuery = NonNullable<operations['listSpotMarkets']['parameters']['query']>;
export type SpotSortField = NonNullable<SpotListQuery['sort']>;
export type SpotSortOrder = NonNullable<SpotListQuery['order']>;
export type MarketExchange = NonNullable<SpotListQuery['exchange']>;

export type CandlesResponse = components['schemas']['CandlesResponse'];
export type Candle = components['schemas']['Candle'];
export type Ticker24h = components['schemas']['Ticker24h'];
export type Supply = components['schemas']['Supply'];
export type AssetHolders = components['schemas']['AssetHolders'];
export type HolderWallet = components['schemas']['HolderWallet'];
export type SpotOrderBook = components['schemas']['SpotOrderBook'];
export type OrderBookLevel = components['schemas']['OrderBookLevel'];
export type OrderBookQuery = NonNullable<operations['getSpotOrderBook']['parameters']['query']>;
export type BookHistoryFlash = {
  side?: string;
  price?: string;
  notional?: string;
  seenFrom?: string;
  seenUntil?: string;
  durationSec?: number;
  brief?: boolean;
};

export type BookHistoryMove = {
  side?: string;
  price?: string;
  fromNotional?: string;
  toNotional?: string;
  peakNotional?: string;
  change?: string;
  seenFrom?: string;
  durationSec?: number;
};

export type BookHistorySnapshot = {
  sampledAt?: string;
  mid?: string;
  flashes?: BookHistoryFlash[];
  moves?: BookHistoryMove[];
};

export type BookHistoryReport = {
  symbol?: string;
  exchange?: string;
  summary?: string;
  note?: string;
  snapshot?: BookHistorySnapshot;
  snapshots?: BookHistorySnapshot[];
};

export type BookHistoryQuery = {
  symbol: string;
  exchange?: string;
  at?: string;
  limit?: number;
};

export type OrderBookHeatmap = components['schemas']['OrderBookHeatmap'];
export type OrderBookHeatmapQuery = NonNullable<
  operations['getSpotOrderBookHeatmap']['parameters']['query']
>;
export type LiquidationHuntHeatmap = components['schemas']['LiquidationHuntHeatmap'];
export type LiquidationHuntHeatmapGrid = components['schemas']['LiquidationHuntHeatmapGrid'];
export type LiquidationHuntHeatmapReview = components['schemas']['LiquidationHuntHeatmapReview'];
export type LiquidationHuntHeatmapReviewVenue = components['schemas']['LiquidationHuntHeatmapReviewVenue'];
export type LiquidationHuntHeatmapReviewHorizon = components['schemas']['LiquidationHuntHeatmapReviewHorizon'];
export type LiquidationHuntHeatmapReviewSignal = components['schemas']['LiquidationHuntHeatmapReviewSignal'];
export type LiquidationHuntHeatmapReviewSignalHorizon =
  components['schemas']['LiquidationHuntHeatmapReviewSignalHorizon'];
export type LiquidationHuntHeatmapQuery = NonNullable<
  operations['getMarketLiquidationHuntHeatmap']['parameters']['query']
>;
export type LiquidationPrintHeatmapQuery = NonNullable<
  operations['getMarketLiquidationPrintHeatmap']['parameters']['query']
>;
export type LiquidationOpenLevels = components['schemas']['LiquidationOpenLevels'];
export type LiquidationOpenLevelBook = components['schemas']['LiquidationOpenLevelBook'];
export type LiquidationOpenLevelSegment = components['schemas']['LiquidationOpenLevelSegment'];
export type LiquidationOpenLevelPoint = components['schemas']['LiquidationOpenLevelPoint'];
export type LiquidationOpenLevelsQuery = NonNullable<
  operations['getMarketLiquidationOpenLevels']['parameters']['query']
>;
export type LiquidationHunt = components['schemas']['LiquidationHunt'];
export type LiquidationHuntQuery = NonNullable<
  operations['getMarketLiquidationHunt']['parameters']['query']
>;
export type LiquidationMaxPain = components['schemas']['LiquidationMaxPain'];
export type LiquidationMaxPainQuery = NonNullable<
  operations['getMarketLiquidationMaxPain']['parameters']['query']
>;

export type CandlesQuery = NonNullable<operations['getCandles']['parameters']['query']>;
export type Ticker24hQuery = NonNullable<operations['getTicker24h']['parameters']['query']>;
export type SupplyQuery = NonNullable<operations['getSupply']['parameters']['query']>;
export type HoldersQuery = NonNullable<operations['getHolders']['parameters']['query']>;
export type AssetProfile = components['schemas']['AssetProfile'];
export type AssetProfileQuery = NonNullable<operations['getAssetProfile']['parameters']['query']>;
export type MarketOpenInterest = components['schemas']['MarketOpenInterest'];
export type MarketBasis = components['schemas']['MarketBasis'];
export type BasisQuery = NonNullable<operations['getMarketBasis']['parameters']['query']>;
export type MarketOptions = components['schemas']['MarketOptions'];
export type MarketOptionsExpiry = components['schemas']['MarketOptionsExpiry'];
export type MarketOptionsExpiryStrike = components['schemas']['MarketOptionsExpiryStrike'];
export type MarketOptionsATM = components['schemas']['MarketOptionsATM'];
export type MarketOptionsVolPoint = components['schemas']['MarketOptionsVolPoint'];
export type MarketOptionsSkew = components['schemas']['MarketOptionsSkew'];
export type MarketOptionsGamma = components['schemas']['MarketOptionsGamma'];
export type MarketOptionsGammaLevel = components['schemas']['MarketOptionsGammaLevel'];
export type MarketOptionsFinding = components['schemas']['MarketOptionsFinding'];
export type MarketOptionsVsMarket = components['schemas']['MarketOptionsVsMarket'];
export type MarketOptionsHistory = components['schemas']['MarketOptionsHistory'];
export type MarketOptionsHistoryWindow = components['schemas']['MarketOptionsHistoryWindow'];
export type MarketOptionsHistoryPoint = components['schemas']['MarketOptionsHistoryPoint'];
export type MarketOptionsMove = components['schemas']['MarketOptionsMove'];
export type MarketOptionsMoves = components['schemas']['MarketOptionsMoves'];
export type MarketOptionsStrikeOI = components['schemas']['MarketOptionsStrikeOI'];
export type MarketOptionsStrikeOIWindow = components['schemas']['MarketOptionsStrikeOIWindow'];
export type MarketOptionsStrikeOIRow = components['schemas']['MarketOptionsStrikeOIRow'];
export type OptionsQuery = NonNullable<operations['getMarketOptions']['parameters']['query']>;
export type MarketFundingRate = components['schemas']['MarketFundingRate'];
export type MarketLiquidations = components['schemas']['MarketLiquidations'];
export type OpenInterestQuery = NonNullable<
  operations['getMarketOpenInterest']['parameters']['query']
>;
export type LiquidationsQuery = NonNullable<
  operations['getMarketLiquidations']['parameters']['query']
>;
export type MarketLiquidationOverview = components['schemas']['MarketLiquidationOverview'];
export type MarketLiquidationCoin = components['schemas']['MarketLiquidationCoin'];
export type LiquidationOverviewQuery = NonNullable<
  operations['getMarketLiquidationOverview']['parameters']['query']
>;
export type MarketLiquidationLevels = components['schemas']['MarketLiquidationLevels'];
export type LiquidationLevelsQuery = NonNullable<
  operations['getMarketLiquidationLevels']['parameters']['query']
>;
export type MarketLiquidationCascade = components['schemas']['MarketLiquidationCascade'];
export type MarketLiquidationCascadeScan = components['schemas']['MarketLiquidationCascadeScan'];
export type LiquidationCascadeQuery = NonNullable<
  operations['getMarketLiquidationCascade']['parameters']['query']
>;
export type LiquidationCascadeScanQuery = NonNullable<
  operations['scanMarketLiquidationCascades']['parameters']['query']
>;
export type CvdQuery = NonNullable<operations['getMarketCVD']['parameters']['query']>;
export type MarketUnusualReading = components['schemas']['MarketUnusualReading'];
export type MarketUnusualVenue = components['schemas']['MarketUnusualVenue'];
export type MarketUnusual = components['schemas']['MarketUnusual'];
export type UnusualQuery = NonNullable<operations['getMarketUnusual']['parameters']['query']>;
export type MarketUnusualScanHit = components['schemas']['MarketUnusualScanHit'];
export type MarketUnusualScan = components['schemas']['MarketUnusualScan'];
export type UnusualScanQuery = NonNullable<operations['scanMarketUnusual']['parameters']['query']>;
export type MarketLongShortRatio = components['schemas']['MarketLongShortRatio'];
export type LongShortQuery = NonNullable<operations['getMarketLongShortRatio']['parameters']['query']>;
export type HistorySpan = components['schemas']['HistorySpan'];
export type HistorySpanQuery = NonNullable<operations['getMarketHistorySpan']['parameters']['query']>;
export type MarketFuturesHistory = components['schemas']['MarketFuturesHistory'];
export type FuturesHistoryQuery = NonNullable<
  operations['getMarketFuturesHistory']['parameters']['query']
>;
export type MarketVolumeShare = components['schemas']['MarketVolumeShare'];
export type VolumeShareQuery = NonNullable<operations['getMarketVolumeShare']['parameters']['query']>;
export type SpotPerpLiquidity = components['schemas']['SpotPerpLiquidity'];
export type SpotPerpBook = components['schemas']['SpotPerpBook'];
export type SpotPerpVenue = components['schemas']['SpotPerpVenue'];
export type SpotPerpQuery = NonNullable<operations['getSpotPerpLiquidity']['parameters']['query']>;
export type MarketPerpDepthOverview = components['schemas']['MarketPerpDepthOverview'];
export type MarketPerpDepthCoin = components['schemas']['MarketPerpDepthCoin'];
export type MarketPerpDepthShareWindow = components['schemas']['MarketPerpDepthShareWindow'];
export type MarketPerpDepthShareDriver = components['schemas']['MarketPerpDepthShareDriver'];
export type MarketPerpDepthOverviewBand = components['schemas']['MarketPerpDepthOverviewBand'];
export type MarketPerpDepthOverviewBandCoin = components['schemas']['MarketPerpDepthOverviewBandCoin'];
export type PerpDepthOverviewQuery = NonNullable<
  operations['getMarketPerpDepthOverview']['parameters']['query']
>;
export type SymbolMarketQuery = { symbol: string; exchange?: string; window?: string };

export type MarketBreadthWindow = {
  window?: string;
  alignment?: string;
  title?: string;
  summary?: string;
  complete?: boolean;
  counts?: {
    up?: number;
    down?: number;
    flat?: number;
    total?: number;
    upPct?: string;
    downPct?: string;
    volumeUpPct?: string;
  };
  btc?: { changePct?: string };
  eth?: { changePct?: string };
};
export type MarketBreadth = {
  exchange?: string;
  quote?: string;
  universe?: number;
  summary?: string;
  note?: string;
  windows?: MarketBreadthWindow[];
};
export type MarketBreadthQuery = { exchange?: string; limit?: number };

export type MarketNetPositionWindow = {
  window?: string;
  netLong?: string;
  netShort?: string;
  buy?: string;
  sell?: string;
  oiDelta?: string;
  unexplained?: string;
  complete?: boolean;
  summary?: string;
};
export type MarketNetPosition = {
  symbol?: string;
  exchange?: string;
  summary?: string;
  note?: string;
  windows?: MarketNetPositionWindow[];
};
export type NetPositionQuery = { symbol: string; exchange?: string };

export type FearGreedFactor = {
  id?: string;
  name?: string;
  detail?: string;
  reading?: number;
  weight?: number;
  sharePct?: number;
  pull?: string;
  toward?: string;
};
export type FearGreedSample = {
  time?: string;
  score?: number;
  level?: string;
  topId?: string;
  topPull?: string;
};
export type FearGreedWindow = {
  window?: string;
  pastScore?: number;
  pastLevel?: string;
  score?: number;
  level?: string;
  change?: number;
  complete?: boolean;
  sampleTime?: string;
};
export type FearGreedReport = {
  score?: number;
  level?: string;
  label?: string;
  asOf?: string;
  used?: number;
  total?: number;
  complete?: boolean;
  top?: FearGreedFactor | null;
  factors?: FearGreedFactor[];
  missing?: string[];
  history?: FearGreedSample[];
  windows?: FearGreedWindow[];
  summary?: string;
  note?: string;
};

export type MarketVolumeSurgeWindow = {
  window?: string;
  current?: string;
  typical?: string;
  ratio?: string;
  grade?: string;
  dominant?: string;
  complete?: boolean;
};
export type MarketVolumeSurgeHit = {
  symbol?: string;
  exchange?: string;
  maxRatio?: string;
  hottest?: string;
  grade?: string;
  dominant?: string;
  summary?: string;
  windows?: MarketVolumeSurgeWindow[];
};
export type MarketVolumeSurgeScan = {
  exchange?: string;
  quote?: string;
  summary?: string;
  note?: string;
  hits?: MarketVolumeSurgeHit[];
};
export type MarketVolumeSurgeScanQuery = {
  exchange?: string;
  quote?: string;
  minRatio?: number;
  limit?: number;
};
export type VolumeProfileBin = {
  price?: string;
  high?: string;
  volume?: string;
  buyVolume?: string;
  sellVolume?: string;
  buyPct?: string;
  sharePct?: string;
  oiOpened?: string;
  oiClosed?: string;
  longLiq?: string;
  shortLiq?: string;
  isPoc?: boolean;
  inValueArea?: boolean;
  isHvn?: boolean;
};

export type VolumeProfileVenue = {
  exchange?: string;
  summary?: string;
  error?: string;
  complete?: boolean;
  started?: string;
  lastPrice?: string;
  tickSize?: string;
  oiOpened?: string;
  oiClosed?: string;
  oiKnown?: boolean;
  longLiq?: string;
  shortLiq?: string;
  liqKnown?: boolean;
  bins?: VolumeProfileBin[];
  poc?: { price?: string; high?: string; volume?: string };
  valueArea?: { low?: string; high?: string };
};

export type VolumeProfileReport = {
  symbol?: string;
  exchange?: string;
  window?: string;
  summary?: string;
  note?: string;
  source?: string;
  definite?: boolean;
  missing?: string[];
  venues?: VolumeProfileVenue[];
  combined?: VolumeProfileVenue;
};

export type FlowAtPricesLevel = {
  price?: string;
  high?: string;
  buyVolume?: string;
  sellVolume?: string;
  volume?: string;
  buyCount?: number;
  sellCount?: number;
  trades?: number;
  delta?: string;
  buyPct?: string;
  sharePct?: string;
  dominant?: string;
  isPoc?: boolean;
};

export type FlowAtPricesVenue = {
  exchange?: string;
  summary?: string;
  error?: string;
  tickSize?: string;
  lastPrice?: string;
  delta?: string;
  truncated?: boolean;
  levels?: FlowAtPricesLevel[];
  poc?: { price?: string; volume?: string; delta?: string; sharePct?: string };
};

export type FlowAtPricesReport = {
  symbol?: string;
  exchange?: string;
  window?: string;
  summary?: string;
  note?: string;
  tickSize?: string;
  venues?: FlowAtPricesVenue[];
  combined?: FlowAtPricesVenue;
};

/** One price row inside a candle. Quantities are API strings. */
export type CandleFootprintRow = {
  price?: string;
  high?: string;
  buyVolume?: string;
  sellVolume?: string;
  delta?: string;
  buyCount?: number;
  sellCount?: number;
  imbalance?: string;
  isPoc?: boolean;
};

/** One candle's buy and sell ladder. `time` is the candle open. */
export type CandleFootprintCandle = {
  time?: string;
  buyVolume?: string;
  sellVolume?: string;
  delta?: string;
  buyCount?: number;
  sellCount?: number;
  poc?: string;
  rows?: CandleFootprintRow[];
};

export type CandleFootprintBook = {
  exchange?: string;
  candles?: CandleFootprintCandle[];
};

export type CandleFootprintReport = {
  symbol?: string;
  exchange?: string;
  interval?: string;
  stepSec?: number;
  from?: string;
  to?: string;
  asOf?: string;
  definite?: boolean;
  truncated?: boolean;
  missing?: string[];
  venues?: CandleFootprintBook[];
  combined?: CandleFootprintBook;
  summary?: string;
  note?: string;
};

export type CandleFootprintQuery = {
  symbol: string;
  exchange?: string;
  interval: string;
  startTime?: string;
  endTime?: string;
};

export type VenueDivergenceReport = {
  symbol?: string;
  alignment?: string;
  title?: string;
  summary?: string;
  note?: string;
  outliers?: string[];
  groups?: { lean?: string; exchanges?: string[] }[];
  venues?: { exchange?: string; lean?: string; regime?: string; oiChange?: string }[];
  diffs?: {
    metric?: string;
    label?: string;
    alignment?: string;
    whyItMatters?: string;
    same?: string[];
    different?: string[];
    readings?: { exchange?: string; value?: string }[];
  }[];
};

export type MarketDeskReport = {
  symbol?: string;
  exchange?: string;
  summary?: string;
  note?: string;
  combined?: {
    summary?: string;
    lean?: string;
    headline?: { below?: { notional?: string }; above?: { notional?: string }; lean?: string };
    windows?: { window?: string; summary?: string; ratio?: string; grade?: string; dominant?: string }[];
  };
  venues?: {
    exchange?: string;
    summary?: string;
    error?: string;
    headline?: {
      lean?: string;
      below?: { notional?: string };
      above?: { notional?: string };
    };
    windows?: { window?: string; summary?: string; ratio?: string; grade?: string; dominant?: string }[];
  }[];
  windows?: { window?: string; summary?: string }[];
  events?: {
    symbol?: string;
    side?: string;
    kind?: string;
    notional?: string;
    intent?: string;
    oiChangePct?: string;
    priceAfterPct?: string;
    oiAfterPct?: string;
    afterReady?: boolean;
  }[];
};
export type MarketParticipation = components['schemas']['MarketParticipation'];
export type ParticipationQuery = NonNullable<operations['getMarketParticipation']['parameters']['query']>;
export type MarketPrintSize = components['schemas']['MarketPrintSize'];
export type PrintSizeQuery = NonNullable<operations['getMarketPrintSize']['parameters']['query']>;
export type PrintSizeSide = {
  avg?: string;
  priorAvg?: string;
  count?: number;
  priorCount?: number;
  ratio?: string;
  kind?: string;
};
export type PrintSizeCoinWindow = {
  window?: string;
  buy?: PrintSizeSide;
  sell?: PrintSizeSide;
  read?: string;
  summary?: string;
};
export type PrintSizeCoin = {
  exchange?: string;
  symbol?: string;
  windows?: PrintSizeCoinWindow[];
  summary?: string;
};
export type PrintSizeMarketWindow = {
  window?: string;
  buy?: PrintSizeSide;
  sell?: PrintSizeSide;
  read?: string;
  largerCoins?: number;
  smallerCoins?: number;
  fewLargeCoins?: number;
  manySmallCoins?: number;
  stableCoins?: number;
  quietCoins?: number;
  coins?: number;
  summary?: string;
};
export type MarketPrintSizeOverview = {
  exchange?: string;
  quote?: string;
  universe?: number;
  asOf?: string;
  windows?: PrintSizeMarketWindow[];
  coins?: PrintSizeCoin[];
  summary?: string;
  note?: string;
  coverage?: MarketVenueCoverage;
  definite?: boolean;
};
export type PrintSizeOverviewQuery = {
  exchange?: string;
  quote?: string;
  limit?: number;
};
export type MarketBotTracker = components['schemas']['MarketBotTracker'];
export type BotCluster = components['schemas']['BotCluster'];
export type BotTrackerQuery = NonNullable<operations['getMarketBotTracker']['parameters']['query']>;
export type MarketOrderCount = components['schemas']['MarketOrderCount'];
export type OrderCountQuery = NonNullable<operations['getMarketOrderCount']['parameters']['query']>;
export type OrderCountWindow = {
  window?: string;
  buyCount?: number;
  sellCount?: number;
  total?: number;
  delta?: number;
  ratio?: string;
  lean?: string;
  complete?: boolean;
  summary?: string;
};
export type OrderCountCoin = {
  exchange?: string;
  symbol?: string;
  windows?: OrderCountWindow[];
  summary?: string;
  error?: string;
};
export type OrderCountMarketWindow = {
  window?: string;
  buyCount?: number;
  sellCount?: number;
  total?: number;
  delta?: number;
  ratio?: string;
  lean?: string;
  buyCoins?: number;
  sellCoins?: number;
  balancedCoins?: number;
  quietCoins?: number;
  coins?: number;
  complete?: boolean;
  summary?: string;
};
export type MarketVenueRead = {
  exchange?: string;
  status?: string;
  asOf?: string;
  ageSec?: number;
};
export type MarketVenueCoverage = {
  complete?: boolean;
  venues?: MarketVenueRead[];
  missing?: string[];
  stale?: string[];
};
export type DerivativesVenue = {
  exchange?: string;
  oiValue?: string;
  oiStatus?: string;
  fundingRate?: string;
  fundingRatePct?: string;
  fundingPayer?: string;
  fundingStatus?: string;
  markPrice?: string;
  liqLong?: string;
  liqShort?: string;
  liqStatus?: string;
  longPct?: string;
  longStatus?: string;
  takerBuyPct?: string;
  takerStatus?: string;
  gapPct?: string;
  gapStatus?: string;
  cvdDay?: string;
  cvdStatus?: string;
  printChangePct?: string;
  printStatus?: string;
  orderRatio?: string;
  orderLean?: string;
  orderBuy?: number;
  orderSell?: number;
  orderStatus?: string;
  unusualMetric?: string;
  unusualBand?: string;
  unusualDistance?: string;
  liqGrade?: string;
  liqHeatStatus?: string;
  bidPct?: string;
  bookStatus?: string;
};
export type DerivativesCoin = {
  symbol?: string;
  base?: string;
  price?: string;
  priceExchange?: string;
  fundingRate?: string;
  fundingRatePct?: string;
  fundingPayer?: string;
  fundingWeight?: string;
  fundingDefinite?: boolean;
  openInterest?: string;
  openInterestDefinite?: boolean;
  change1h?: string;
  change1hPct?: string;
  change1hDefinite?: boolean;
  change24h?: string;
  change24hPct?: string;
  change24hDefinite?: boolean;
  liqLong?: string;
  liqShort?: string;
  liqDefinite?: boolean;
  longPct?: string;
  longDefinite?: boolean;
  takerBuyPct?: string;
  takerDefinite?: boolean;
  gapPct?: string;
  gapDefinite?: boolean;
  gapScope?: string;
  cvdDay?: string;
  cvdDayDefinite?: boolean;
  printChangePct?: string;
  printRead?: string;
  printDefinite?: boolean;
  orderRatio?: string;
  orderLean?: string;
  orderBuy?: number;
  orderSell?: number;
  orderDefinite?: boolean;
  unusualMetric?: string;
  unusualBand?: string;
  unusualDistance?: string;
  unusualPercentile?: string;
  unusualVenue?: string;
  unusualDefinite?: boolean;
  liqHeat?: string;
  liqHeatRatio?: string;
  liqHeatVenue?: string;
  liqHeatDefinite?: boolean;
  bidPct?: string;
  bidDefinite?: boolean;
  venues?: DerivativesVenue[];
  coverage?: MarketVenueCoverage;
  recorded?: { kind?: string; exchange?: string; since?: string; spanSec?: number }[];
};
export type MarketDerivativesBoard = {
  exchange?: string;
  universe?: number;
  asOf?: string;
  coins?: DerivativesCoin[];
  coverage?: MarketVenueCoverage;
  definite?: boolean;
  summary?: string;
  note?: string;
};
export type DerivativesBoardQuery = {
  exchange?: string;
  limit?: number;
};
export type MarketOrderCountOverview = {
  exchange?: string;
  quote?: string;
  universe?: number;
  asOf?: string;
  windows?: OrderCountMarketWindow[];
  coins?: OrderCountCoin[];
  summary?: string;
  note?: string;
  coverage?: MarketVenueCoverage;
  definite?: boolean;
};
export type OrderCountOverviewQuery = {
  exchange?: string;
  quote?: string;
  limit?: number;
};
export type ImpactScanHit = {
  symbol?: string;
  scope?: string;
  side?: string;
  slippagePct?: number;
  slippageVsBestPct?: number;
  impactPct?: number;
  impactAvailable?: boolean;
  exhausted?: boolean;
  averagePrice?: string;
  bestPrice?: string;
  spentNotional?: string;
  summary?: string;
};
export type ImpactScan = {
  exchange?: string;
  quote?: string;
  side?: string;
  notional?: number;
  universe?: number;
  medianSlippagePct?: number;
  exhaustedCoins?: number;
  hits?: ImpactScanHit[];
  summary?: string;
  note?: string;
};
export type ImpactScanQuery = {
  exchange?: string;
  quote?: string;
  side?: string;
  notional?: number;
  limit?: number;
};
export type WallScanHit = {
  exchange?: string;
  symbol?: string;
  side?: string;
  price?: string;
  notional?: string;
  distancePct?: string;
  share?: number;
  kind?: string;
  appearCount?: number;
  present?: boolean;
};
export type WallScan = {
  exchange?: string;
  quote?: string;
  universe?: number;
  suspicious?: number;
  flickers?: number;
  hits?: WallScanHit[];
  summary?: string;
  note?: string;
};
export type WallScanQuery = {
  exchange?: string;
  quote?: string;
  limit?: number;
};
export type BookGap = {
  side?: string;
  direction?: string;
  fromPrice?: string;
  toPrice?: string;
  fromPct?: number;
  toPct?: number;
  widthPct?: number;
  notional?: string;
};
export type BookGaps = {
  symbol?: string;
  exchange?: string;
  midPrice?: string;
  rangePct?: number;
  easier?: string;
  upNotional?: string;
  downNotional?: string;
  upGapPct?: number;
  downGapPct?: number;
  gaps?: BookGap[];
  summary?: string;
  note?: string;
};
export type BookGapsQuery = {
  symbol: string;
  exchange?: string;
  rangePct?: number;
};
export type BookGapScanHit = {
  symbol?: string;
  scope?: string;
  easier?: string;
  upNotional?: string;
  downNotional?: string;
  upGapPct?: number;
  downGapPct?: number;
  widestPct?: number;
  summary?: string;
};
export type BookGapScan = {
  exchange?: string;
  quote?: string;
  rangePct?: number;
  universe?: number;
  hits?: BookGapScanHit[];
  summary?: string;
  note?: string;
};
export type SweepRecoveryRow = {
  exchange?: string;
  symbol?: string;
  side?: string;
  level?: string;
  beforeNotional?: string;
  troughNotional?: string;
  backNotional?: string;
  recovered?: boolean;
  recoverySeconds?: number;
  recoveredPct?: number;
  summary?: string;
};
export type SweepRecoveryReport = {
  symbol?: string;
  exchange?: string;
  medianRecoverySeconds?: number;
  recovered?: number;
  pending?: number;
  rows?: SweepRecoveryRow[];
  summary?: string;
  note?: string;
};
export type BookGapScanQuery = {
  exchange?: string;
  quote?: string;
  rangePct?: number;
  limit?: number;
};
export type WideSpreadHit = {
  exchange?: string;
  symbol?: string;
  base?: string;
  currentPct?: string;
  typicalPct?: string;
  ratio?: string;
  samples?: number;
};
export type WideSpreadScan = {
  exchange?: string;
  quote?: string;
  known?: number;
  ready?: number;
  hits?: WideSpreadHit[];
  summary?: string;
  note?: string;
};
export type WideSpreadScanQuery = {
  exchange?: string;
  quote?: string;
  limit?: number;
};
export type WideSpread = {
  symbol?: string;
  exchange?: string;
  currentPct?: string;
  typicalPct?: string;
  ratio?: string;
  samples?: number;
  opened?: boolean;
  summary?: string;
  note?: string;
};
export type WideSpreadQuery = {
  symbol: string;
  exchange?: string;
};
export type MarketPressure = components['schemas']['MarketPressure'];
export type PressureQuery = NonNullable<operations['getMarketPressure']['parameters']['query']>;
export type MarketForensicsQuery = NonNullable<
  operations['getMarketForensics']['parameters']['query']
>;
export type MarketForensics = components['schemas']['MarketForensics'];
export type MarketForensicsScanQuery = NonNullable<
  operations['scanMarketForensics']['parameters']['query']
>;
export type MarketForensicsScan = components['schemas']['MarketForensicsScan'];
export type MarketForensicsRadarQuery = {
  exchange?: string;
  symbol?: string;
  since?: string;
  limit?: number;
};

export type MarketCvdPoint = {
  time?: string;
  price?: string;
  delta?: string;
  cvd?: string;
  buyNotional?: string;
  sellNotional?: string;
};

export type MarketCvdVenue = {
  exchange?: string;
  lastCvd?: string;
  lastPrice?: string;
  summary?: string;
  error?: string;
  complete?: boolean;
  points?: MarketCvdPoint[];
  windows?: { window?: string; cvdChange?: string; label?: string; complete?: boolean }[];
  anchors?: {
    anchor?: string;
    from?: string;
    cvd?: string;
    priceChangePct?: string;
    summary?: string;
    complete?: boolean;
  }[];
};

export type MarketCvd = {
  symbol?: string;
  exchange?: string;
  /** 24h | 7d | 30d | 90d */
  range?: string;
  /** 5m | 30m | 2h | 4h */
  bar?: string;
  summary?: string;
  note?: string;
  venues?: MarketCvdVenue[];
  combined?: MarketCvdVenue;
  spotCombined?: MarketCvdVenue;
  spotFutures?: { summary?: string };
};
export type IntervalsQuery = NonNullable<operations['listIntervals']['parameters']['query']>;
export type IndicatorsQuery = NonNullable<operations['getIndicators']['parameters']['query']>;
export type PumpEventsQuery = NonNullable<operations['getPumpEvents']['parameters']['query']>;
export type ScanPumpEventsQuery = NonNullable<
  operations['scanPumpEvents']['parameters']['query']
>;
export type RSIHeatmapQuery = NonNullable<operations['getRSIHeatmap']['parameters']['query']>;
export type RSIHeatmapResponse = components['schemas']['RSIHeatmap'];

export type IntervalsResponse = {
  exchange: string;
  intervals: string[];
};

/** Prefer OpenAPI 200 body when present; fall back to stable hand shape. */
export type IndicatorsResponse = {
  exchange?: string;
  symbol?: string;
  interval?: string;
  rsiPeriod?: number;
  emaPeriods?: number[];
  latest?: {
    rsi?: number | null;
    zone?: 'oversold' | 'overbought' | 'neutral' | '';
    ema?: Record<string, number>;
  };
  points?: {
    openTime?: string;
    close?: number;
    rsi?: number | null;
    ema?: Record<string, number>;
  }[];
  note?: string;
};

export type ExchangesResponse = {
  exchanges: string[];
  default: string;
  venueQuotes?: Record<string, string>;
  marketCapQuotes?: Record<string, string>;
};

export type FxRatesResponse = {
  base?: string;
  asOf?: string;
  rates?: Record<string, number>;
  venueQuotes?: Record<string, string>;
  marketCapQuotes?: Record<string, string>;
  aliases?: Record<string, string>;
  stale?: boolean;
  note?: string;
};

export type ProductTagsResponse = {
  exchange: string;
  tags: string[];
};

/** OpenAPI-generated pump shapes (schema.d.ts). */
export type PumpEventDto = components['schemas']['PumpEvent'];
export type PumpEventsResponse = components['schemas']['PumpEventsResponse'];
export type PumpScanHitDto = components['schemas']['PumpScanHit'];
export type ScanPumpEventsResponse = components['schemas']['PumpScanResponse'];

export type DelistScheduleResponse = components['schemas']['DelistScheduleResponse'];
export type DelistScheduleItem = components['schemas']['DelistScheduleItem'];
export type PostDelistResponse = components['schemas']['PostDelistResponse'];
export type PostDelistQuery = NonNullable<operations['getPostDelist']['parameters']['query']>;
