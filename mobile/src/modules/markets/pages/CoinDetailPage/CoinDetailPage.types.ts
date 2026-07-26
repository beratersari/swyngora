import type { CandleChartOverlay } from '@/components/organisms/CandleChart';
import type { ChartCandle, ChartLinePoint } from '@/libs/utils';

export type CoinDetailPageViewModel = {
  symbol: string;
  exchange: string;

  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  headerLoading: boolean;

  statsItems: { label: string; value: string }[];
  statsLoading: boolean;
  tickerError: string | null;
  supplyError: string | null;

  intervals: string[];
  intervalsLoading: boolean;
  interval: string;
  onSelectInterval: (interval: string) => void;
  showEma: boolean;
  onToggleEma: () => void;

  candles: ChartCandle[];
  candleOverlays: CandleChartOverlay[];
  candlesLoading: boolean;
  candlesError: string | null;

  rsiPoints: ChartLinePoint[];
  latestRsi: number | null;
  indicatorsLoading: boolean;
  indicatorsError: string | null;
  emaLatestLabels: string[];

  onBack: () => void;
  onRetry: () => void;
};

export type CoinDetailPageProps = {
  viewModel?: CoinDetailPageViewModel;
};
