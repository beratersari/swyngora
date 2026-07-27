import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { WatchlistPage } from './WatchlistPage';
import type { WatchlistPageViewModel } from './WatchlistPage.types';

const baseVm = (): WatchlistPageViewModel => ({
  title: 'Favorites',
  countLabel: null,
  isLoading: false,
  isRefreshing: false,
  isPollingPaused: true,
  errorMessage: null,
  emptyMessage: 'No favorites yet — open Markets and tap ★ on a pair',
  actionError: null,
  indicatorsError: null,
  indicatorsDisclaimer: null,
  pairs: [],
  onRetry: vi.fn(),
  onRefresh: vi.fn(),
  onOpenMarkets: vi.fn(),
  onPressRow: vi.fn(),
  onUnstar: vi.fn(),
  pollQuotes: false,
  rsiByKey: new Map(),
});

describe('WatchlistPage (Favorites)', () => {
  it('renders empty state with CTA', () => {
    renderWithProviders(<WatchlistPage viewModel={baseVm()} />);
    expect(screen.getByText('Favorites')).toBeTruthy();
    expect(
      screen.getByText('No favorites yet — open Markets and tap ★ on a pair'),
    ).toBeTruthy();
    expect(screen.getByText('Open Markets')).toBeTruthy();
  });

  it('renders error with retry', () => {
    renderWithProviders(
      <WatchlistPage
        viewModel={{
          ...baseVm(),
          emptyMessage: null,
          errorMessage: 'Network error',
        }}
      />,
    );
    expect(screen.getByText('Network error')).toBeTruthy();
    expect(screen.getByText('Retry')).toBeTruthy();
  });

  it('shows indicators error and disclaimer without emptying list chrome', () => {
    renderWithProviders(
      <WatchlistPage
        viewModel={{
          ...baseVm(),
          emptyMessage: null,
          countLabel: '1 favorite',
          pairs: [{ exchange: 'binance', symbol: 'BTCUSDT' }],
          indicatorsError: 'Request failed (502)',
          indicatorsDisclaimer: 'RSI/EMA are informational only — not financial advice.',
        }}
      />,
    );
    expect(screen.getByText(/Indicators: Request failed/)).toBeTruthy();
    expect(screen.getByText(/informational only/)).toBeTruthy();
  });
});
