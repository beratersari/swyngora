import type { CandleChartOverlay } from '@/components/organisms/candle-chart';
import type { PumpEventRowViewModel } from '@/components/organisms/pump-event-list';
import type {
  ChartCandle,
  ChartLinePoint,
  ChartMarker,
  ChartPriceLine,
  CrossExchangeRowModel,
} from '@/libs/utils';

export type CoinDetailPageViewModel = {
  symbol: string;
  exchange: string;

  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  headerLoading: boolean;
  watched: boolean;
  onStarPress: () => void;
  actionError: string | null;

  statsItems: { label: string; value: string }[];
  statsLoading: boolean;
  tickerError: string | null;
  supplyError: string | null;

  crossExchangeTitle: string;
  crossExchangeRows: CrossExchangeRowModel[];
  crossExchangeDisclaimer: string | null;
  crossExchangeUnavailableLabel: string;
  crossExchangeSourceLabel: string;
  crossExchangeCheapestLabel: string;
  crossExchangeCheapestId: string | null;
  onPressCrossExchangeRow: (exchange: string, symbol: string) => void;

  intervals: string[];
  intervalsLoading: boolean;
  interval: string;
  onSelectInterval: (interval: string) => void;
  showEma: boolean;
  onToggleEma: () => void;
  showPumps: boolean;
  onTogglePumps: () => void;
  showPumpMargin: boolean;
  onTogglePumpMargin: () => void;

  candles: ChartCandle[];
  candleOverlays: CandleChartOverlay[];
  chartMarkers: ChartMarker[];
  chartPriceLines: ChartPriceLine[];
  candlesLoading: boolean;
  /** Loading bars while panning left (history). */
  candlesLoadingOlder: boolean;
  candlesError: string | null;
  chartSeriesKey: string;
  canLoadOlderHistory: boolean;
  historyEdgeBars: number;
  onRequestOlderHistory: () => void;

  rsiPoints: ChartLinePoint[];
  latestRsi: number | null;
  indicatorsLoading: boolean;
  indicatorsError: string | null;
  emaLatestLabels: string[];

  pumpEventRows: PumpEventRowViewModel[];
  pumpEventsLoading: boolean;
  pumpEventsError: string | null;
  pumpEventsSubtitle: string | null;
  pumpDisclaimer: string | null;

  onBack: () => void;
  onRetry: () => void;
  askAiLabel: string;
  onAskAi: () => void;
};

export type CoinDetailPageProps = {
  viewModel?: CoinDetailPageViewModel;
};
