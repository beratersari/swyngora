/** Public VM surface consumed by the View (and tests). */
export type HomePageViewModel = {
  title: string;
  apiBaseUrlLabel: string;
  healthStatus: 'unknown' | 'ok' | 'error';
  healthDetail: string | null;
  isLoading: boolean;
  isPollingPaused: boolean;
  errorMessage: string | null;
  onRetry: () => void;
  onOpenMarkets: () => void;
  onOpenPumps: () => void;
  onOpenAsk: () => void;
};

export type HomePageProps = {
  /** Optional injection for tests. Production path omits this. */
  viewModel?: HomePageViewModel;
};
