import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MarketsPage } from './MarketsPage';
import type { MarketsPageViewModel } from './MarketsPage.types';

function makeVm(overrides: Partial<MarketsPageViewModel> = {}): MarketsPageViewModel {
  return {
    title: 'Markets',
    exchanges: ['binance', 'coinbase'],
    selectedExchange: 'binance',
    onSelectExchange: vi.fn(),
    exchangesLoading: false,
    search: '',
    onSearchChange: vi.fn(),
    isSearchDebouncing: false,
    activeFilterCount: 0,
    filterSummary: null,
    onOpenFilters: vi.fn(),
    rows: [
      {
        id: 'BTCUSDT',
        symbol: 'BTCUSDT',
        lastPriceLabel: '67,000',
        changePercentLabel: '+1.50%',
        changeTone: 'success',
        quoteVolumeLabel: '1.50B',
        marketCapLabel: '1.30T',
        tagsLabel: 'Layer1',
      },
    ],
    total: 1,
    hasMore: false,
    isLoading: false,
    isLoadingMore: false,
    isRefreshing: false,
    isPollingPaused: false,
    errorMessage: null,
    emptyMessage: null,
    summaryLabel: 'Showing 1 of 1',
    detailHint: null,
    onLoadMore: vi.fn(),
    onRetry: vi.fn(),
    onRefresh: vi.fn(),
    onPressRow: vi.fn(),
    ...overrides,
  };
}

describe('MarketsPage', () => {
  it('renders list chrome with filters button', () => {
    render(<MarketsPage viewModel={makeVm()} />);
    expect(screen.getByText('Markets')).toBeTruthy();
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    expect(screen.getByText('Filters')).toBeTruthy();
    expect(screen.queryByText('Quote')).toBeNull();
  });

  it('shows active filter count', () => {
    render(
      <MarketsPage
        viewModel={makeVm({
          activeFilterCount: 2,
          filterSummary: 'USD · 3 tags',
        })}
      />,
    );
    expect(screen.getByText('Filters (2)')).toBeTruthy();
    expect(screen.getByText(/Active: USD/)).toBeTruthy();
  });

  it('renders empty and error states', () => {
    const { rerender } = render(
      <MarketsPage
        viewModel={makeVm({
          rows: [],
          emptyMessage: 'No markets match filters',
          summaryLabel: null,
        })}
      />,
    );
    expect(screen.getByText('No markets match filters')).toBeTruthy();
    rerender(
      <MarketsPage
        viewModel={makeVm({
          rows: [],
          errorMessage: 'Network error',
          emptyMessage: null,
        })}
      />,
    );
    expect(screen.getByText('Network error')).toBeTruthy();
  });
});
