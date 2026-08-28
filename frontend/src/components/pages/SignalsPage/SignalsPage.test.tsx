import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { SignalsPage } from './SignalsPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetWatchlistQuery: () => ({
      data: { items: [{ exchange: 'binance', symbol: 'BTCUSDT' }] },
      isLoading: false,
      isError: false,
    }),
    useListIntervalsQuery: () => ({
      data: { intervals: ['1h', '4h'] },
      isLoading: false,
      isError: false,
    }),
    useListScannerRulesQuery: () => ({
      data: { rules: [], count: 0 },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useListScannerResultsQuery: () => ({
      data: { results: [], count: 0 },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useListScannerBacktestsQuery: () => ({
      data: { backtests: [], count: 0 },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useListScannerBacktestSignalsQuery: () => ({
      data: { signals: [] },
      isFetching: false,
    }),
    useCreateScannerRuleMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useUpdateScannerRuleMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useDeleteScannerRuleMutation: () => [vi.fn(), { isLoading: false }],
    useStartScannerBacktestMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useCancelScannerBacktestMutation: () => [vi.fn(), { isLoading: false }],
    useLazyListSpotMarketsQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
  };
});

describe('SignalsPage', () => {
  it('renders desk title and setup tab', async () => {
    renderWithProviders(<SignalsPage />, { routerEntries: ['/signals'] });
    expect(await screen.findByRole('heading', { name: /swing signals|salınım sinyalleri/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /setups|kurulumlar/i })).toBeInTheDocument();
  });
});
