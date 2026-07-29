import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { HomePage } from './HomePage';
import type { HomePageViewModel } from './HomePage.types';

const stubVm = (): HomePageViewModel => ({
  title: 'Home',
  intro: 'Market snapshot',
  quickActions: [
    { id: 'm', label: 'Markets', onPress: vi.fn() },
    { id: 'p', label: 'Pumps', onPress: vi.fn() },
    { id: 'a', label: 'Ask', onPress: vi.fn() },
  ],
  favorites: [],
  favoritesLoading: false,
  favoritesEmpty: 'No favorites yet — star pairs on Markets.',
  favoritesTitle: 'Favorites',
  movers: [
    {
      id: 'binance|BTCUSDT',
      exchange: 'binance',
      symbol: 'BTCUSDT',
      lastPriceLabel: '67,000',
      changePercentLabel: '+2.00%',
      changeTone: 'success',
      metaLabel: '$1.2B',
    },
  ],
  moversLoading: false,
  moversError: null,
  moversEmpty: null,
  moversTitle: 'Top movers (24h)',
  onOpenMoversSeeAll: vi.fn(),
  onRetryMovers: vi.fn(),
  volume: [],
  volumeLoading: false,
  volumeError: null,
  volumeEmpty: 'No volume data',
  volumeTitle: 'Highest volume',
  onOpenVolumeSeeAll: vi.fn(),
  onRetryVolume: vi.fn(),
  pumps: [
    {
      id: 'p1',
      exchange: 'binance',
      symbol: 'PEPEUSDT',
      returnLabel: '+12%',
      returnTone: 'success',
      metaLabel: '15m · 2 evt',
    },
  ],
  pumpsLoading: false,
  pumpsError: null,
  pumpsEmpty: null,
  pumpsTitle: 'Pump radar',
  pumpsDisclaimer: 'Not a trade signal',
  onRetryPumps: vi.fn(),
  categoriesTitle: 'Categories',
  categoryTags: ['Meme', 'AI'],
  categoriesLoading: false,
  categoriesError: null,
  categoriesEmpty: null,
  onSelectCategory: vi.fn(),
  onOpenCategories: vi.fn(),
  onRetryCategories: vi.fn(),
  formatCategoryLabel: (t) => t,
  seeAllLabel: 'See all',
  retryLabel: 'Retry',
  isRefreshing: false,
  isPollingPaused: true,
  pollingCaption: 'Refresh paused',
  healthStatus: 'ok',
  healthDetail: 'ok',
  apiBaseUrlLabel: '(proxy)',
  onRefresh: vi.fn(),
  onOpenMarkets: vi.fn(),
  onOpenPumps: vi.fn(),
  onOpenAsk: vi.fn(),
  onPressMarket: vi.fn(),
  onPressPump: vi.fn(),
  onOpenFavorites: vi.fn(),
});

describe('HomePage dashboard', () => {
  it('renders widgets from injected view model', () => {
    render(<HomePage viewModel={stubVm()} />);
    expect(screen.getByText('Home')).toBeTruthy();
    expect(screen.getByText('Top movers (24h)')).toBeTruthy();
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    expect(screen.getByText('PEPEUSDT')).toBeTruthy();
    expect(screen.getByText(/No favorites yet/)).toBeTruthy();
    expect(screen.getByText('Categories')).toBeTruthy();
    expect(screen.getByText('Meme')).toBeTruthy();
  });

  it('shows section error with retry', () => {
    render(
      <HomePage
        viewModel={{
          ...stubVm(),
          movers: [],
          moversError: 'Failed to load movers',
        }}
      />,
    );
    expect(screen.getByText('Failed to load movers')).toBeTruthy();
  });
});
