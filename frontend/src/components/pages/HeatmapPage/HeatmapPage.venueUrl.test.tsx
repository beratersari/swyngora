import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useLocation } from 'react-router-dom';
import { renderWithProviders } from '@/test/render';
import { HeatmapPage } from './HeatmapPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListSpotMarketsQuery: () => ({
      data: { items: [{ symbol: 'BTC-USD', lastPrice: '100', priceChangePercent: '1' }] },
      currentData: { items: [{ symbol: 'BTC-USD', lastPrice: '100', priceChangePercent: '1' }] },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
  };
});

vi.mock('@/libs/realtime', () => ({
  usePriceSubscription: () => ({ connected: true }),
  useRealtimeConnected: () => true,
}));

function LocationProbe() {
  const loc = useLocation();
  return <div data-testid="loc">{loc.pathname + loc.search}</div>;
}

describe('HeatmapPage venue URL', () => {
  it('writes the selected venue to the URL so jump search / refresh keep Coinbase', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <>
        <HeatmapPage />
        <LocationProbe />
      </>,
      { routerEntries: ['/heatmap'] },
    );

    await user.click(screen.getByRole('tab', { name: /coinbase/i }));
    expect(screen.getByTestId('loc').textContent).toMatch(/exchange=coinbase/);
  });
});
