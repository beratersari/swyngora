import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { LeaderboardsPage } from './LeaderboardsPage';
import type { LeaderboardsPageViewModel } from './LeaderboardsPage.types';

const stubVm = (): LeaderboardsPageViewModel => ({
  title: 'Leaderboards',
  board: 'gainers',
  boardOptions: [
    { value: 'gainers', label: 'Gainers' },
    { value: 'losers', label: 'Losers' },
    { value: 'volume', label: 'Volume' },
  ],
  onSelectBoard: vi.fn(),
  exchanges: ['binance', 'coinbase'],
  selectedExchange: 'binance',
  onSelectExchange: vi.fn(),
  exchangesLoading: false,
  quote: 'USDT',
  quoteOptions: ['USDT', 'USD'],
  onSelectQuote: vi.fn(),
  rows: [
    {
      id: 'BTCUSDT',
      symbol: 'BTCUSDT',
      lastPriceLabel: '67,000',
      changePercentLabel: '+5.00%',
      changeTone: 'success',
      quoteVolumeLabel: '1.2B',
      marketCapLabel: '1T',
      tagsLabel: '',
      rankLabel: '#1',
    },
  ],
  isLoading: false,
  isLoadingMore: false,
  isRefreshing: false,
  hasMore: false,
  isPollingPaused: true,
  errorMessage: null,
  emptyMessage: null,
  summaryLabel: 'Showing 1 of 1',
  onLoadMore: vi.fn(),
  onRetry: vi.fn(),
  onRefresh: vi.fn(),
  onPressRow: vi.fn(),
  onBack: vi.fn(),
  backLabel: 'Back',
  retryLabel: 'Retry',
});

describe('LeaderboardsPage', () => {
  it('renders boards and ranked row', () => {
    const vm = stubVm();
    render(<LeaderboardsPage viewModel={vm} />);
    expect(screen.getByText('Leaderboards')).toBeTruthy();
    expect(screen.getByText('Gainers')).toBeTruthy();
    expect(screen.getByText('#1')).toBeTruthy();
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    fireEvent.click(screen.getByText('Losers'));
    expect(vm.onSelectBoard).toHaveBeenCalledWith('losers');
  });
});
