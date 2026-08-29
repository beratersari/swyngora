import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { HeatmapPage } from './HeatmapPage';

const refetch = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListSpotMarketsQuery: () => ({
      data: { items: [] },
      currentData: { items: [] },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetRSIHeatmapQuery: () => ({
      data: undefined,
      currentData: undefined,
      isLoading: false,
      isFetching: false,
      isError: true,
      error: { status: 502, data: { error: { message: 'upstream' } } },
      refetch,
    }),
  };
});

vi.mock('@/libs/realtime', () => ({
  usePriceSubscription: () => ({ connected: false }),
  useRealtimeConnected: () => false,
}));

describe('HeatmapPage RSI error', () => {
  it('shows retry when the RSI map fails to load', async () => {
    renderWithProviders(<HeatmapPage />, { routerEntries: ['/heatmap?view=rsi'] });
    expect(await screen.findByText('Could not load RSI heatmap')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: /retry/i }));
    expect(refetch).toHaveBeenCalled();
  });
});
