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
    quote: 'USDT',
    quoteOptions: ['USDT', 'USD'],
    onQuoteChange: vi.fn(),
    availableTags: [],
    selectedTags: [],
    onToggleTag: vi.fn(),
    onClearTags: vi.fn(),
    sort: 'quoteVolume',
    order: 'desc',
    sortOptions: [{ value: 'quoteVolume', label: 'Quote vol' }],
    onSortChange: vi.fn(),
    onOrderChange: vi.fn(),
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
    offset: 0,
    limit: 30,
    onNextPage: vi.fn(),
    onPrevPage: vi.fn(),
    canNext: false,
    canPrev: false,
    isLoading: false,
    isRefreshing: false,
    isPollingPaused: false,
    errorMessage: null,
    emptyMessage: null,
    lastUpdatedLabel: null,
    summaryLabel: '1–1 of 1',
    onRetry: vi.fn(),
    onRefresh: vi.fn(),
    onPressRow: vi.fn(),
    ...overrides,
  };
}

describe('MarketsPage', () => {
  it('renders rows from injected view model', () => {
    render(<MarketsPage viewModel={makeVm()} />);
    expect(screen.getByText('Markets')).toBeTruthy();
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    expect(screen.getByText('+1.50%')).toBeTruthy();
  });

  it('renders empty state', () => {
    render(
      <MarketsPage
        viewModel={makeVm({
          rows: [],
          emptyMessage: 'No markets match filters',
          summaryLabel: '0 results',
        })}
      />,
    );
    expect(screen.getByText('No markets match filters')).toBeTruthy();
  });

  it('renders error with retry', () => {
    render(
      <MarketsPage
        viewModel={makeVm({
          rows: [],
          errorMessage: 'Network error',
          emptyMessage: null,
        })}
      />,
    );
    expect(screen.getByText('Network error')).toBeTruthy();
    expect(screen.getByText('Retry')).toBeTruthy();
  });
});
