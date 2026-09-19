import type {
  DashboardMarketRow,
  DashboardPumpTeaser,
} from '@/libs/utils';

export type HomePageViewModel = {
  title: string;
  intro: string;

  quickActions: { id: string; label: string; onPress: () => void }[];

  favorites: DashboardMarketRow[];
  favoritesLoading: boolean;
  favoritesEmpty: string | null;
  favoritesTitle: string;

  movers: DashboardMarketRow[];
  moversLoading: boolean;
  moversError: string | null;
  moversEmpty: string | null;
  moversTitle: string;
  onOpenMoversSeeAll: () => void;
  onRetryMovers: () => void;

  volume: DashboardMarketRow[];
  volumeLoading: boolean;
  volumeError: string | null;
  volumeEmpty: string | null;
  volumeTitle: string;
  onOpenVolumeSeeAll: () => void;
  onRetryVolume: () => void;

  pumps: DashboardPumpTeaser[];
  pumpsLoading: boolean;
  pumpsError: string | null;
  pumpsEmpty: string | null;
  pumpsTitle: string;
  pumpsDisclaimer: string | null;
  onRetryPumps: () => void;

  categoriesTitle: string;
  categoryTags: string[];
  categoriesLoading: boolean;
  categoriesError: string | null;
  categoriesEmpty: string | null;
  onSelectCategory: (tag: string) => void;
  onOpenCategories: () => void;
  onRetryCategories: () => void;
  formatCategoryLabel: (tag: string) => string;

  seeAllLabel: string;
  retryLabel: string;
  isRefreshing: boolean;
  isPollingPaused: boolean;
  pollingCaption: string | null;

  healthStatus: 'unknown' | 'ok' | 'error';
  healthDetail: string | null;
  apiBaseUrlLabel: string;

  onRefresh: () => void;
  onOpenMarkets: () => void;
  onOpenPumps: () => void;
  onOpenAsk: () => void;
  onPressMarket: (exchange: string, symbol: string) => void;
  onPressPump: (exchange: string, symbol: string) => void;
  onOpenFavorites: () => void;
};

export type HomePageProps = {
  viewModel?: HomePageViewModel;
};
