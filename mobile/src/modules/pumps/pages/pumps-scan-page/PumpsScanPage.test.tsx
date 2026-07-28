import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { PumpsScanPage } from './PumpsScanPage';
import type { PumpsScanPageViewModel } from './PumpsScanPage.types';

const baseVm = (): PumpsScanPageViewModel => ({
  title: 'Pumps',
  exchanges: ['binance'],
  selectedExchange: 'binance',
  onSelectExchange: vi.fn(),
  exchangesLoading: false,
  lookbackHours: 24,
  lookbackOptions: [6, 12, 24, 48],
  onSelectLookback: vi.fn(),
  minReturnPct: 8,
  thresholdOptions: [5, 8, 10],
  onSelectThreshold: vi.fn(),
  direction: 'up',
  directionOptions: [
    { value: 'up', label: 'Up' },
    { value: 'down', label: 'Down' },
  ],
  onSelectDirection: vi.fn(),
  summaryLabel: 'USDT · 15m',
  disclaimer: 'Informational only',
  rows: [],
  isLoading: false,
  isRefreshing: false,
  errorMessage: null,
  emptyMessage: 'No pumps matched — try lower threshold or longer lookback',
  onRetry: vi.fn(),
  onRefresh: vi.fn(),
  onPressRow: vi.fn(),
});

describe('PumpsScanPage', () => {
  it('renders empty state and disclaimer', () => {
    renderWithProviders(<PumpsScanPage viewModel={baseVm()} />);
    expect(screen.getByText('Pumps')).toBeTruthy();
    expect(screen.getByText(/No pumps matched/)).toBeTruthy();
    expect(screen.getByText('Informational only')).toBeTruthy();
  });

  it('renders error with retry', () => {
    renderWithProviders(
      <PumpsScanPage
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

  it('renders hit rows', () => {
    renderWithProviders(
      <PumpsScanPage
        viewModel={{
          ...baseVm(),
          emptyMessage: null,
          rows: [
            {
              id: 'binance|BTCUSDT',
              symbol: 'BTCUSDT',
              exchange: 'binance',
              bestReturnLabel: '+12.00%',
              bestReturnTone: 'success',
              eventsLabel: '2 events',
              metaLabel: '15m',
            },
          ],
        }}
      />,
    );
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    expect(screen.getByText('+12.00%')).toBeTruthy();
  });
});
