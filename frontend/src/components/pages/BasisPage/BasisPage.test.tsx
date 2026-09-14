import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { BasisPage } from './BasisPage';

vi.mock('@/libs/api', async () => {
  const actual = await vi.importActual<typeof import('@/libs/api')>('@/libs/api');
  return {
    ...actual,
    useGetMarketBasisQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        venues: [
          {
            exchange: 'binance',
            last: { kind: 'premium', deltaPct: '+0.10' },
            summary: 'Binance: perpetual is a premium.',
            curve: {
              complete: true,
              shape: 'contango',
              slope: 'up',
              richer: 'far',
              summary: 'Far expiry is more expensive than near.',
              points: [{ symbol: 'BTCUSDT', kind: 'perp', label: 'Perp', mark: '64000' }],
            },
          },
        ],
      },
      currentData: {
        symbol: 'BTCUSDT',
        venues: [
          {
            exchange: 'binance',
            last: { kind: 'premium', deltaPct: '+0.10' },
            summary: 'Binance: perpetual is a premium.',
            curve: {
              complete: true,
              shape: 'contango',
              slope: 'up',
              richer: 'far',
              summary: 'Far expiry is more expensive than near.',
              points: [{ symbol: 'BTCUSDT', kind: 'perp', label: 'Perp', mark: '64000' }],
            },
          },
        ],
      },
      isLoading: false,
      isFetching: false,
      isError: false,
    }),
  };
});

describe('BasisPage', () => {
  it('renders the basis desk', () => {
    renderWithProviders(<BasisPage />, { routerEntries: ['/basis?symbol=BTCUSDT'] });
    expect(screen.getByTestId('basis-panel')).toBeInTheDocument();
    expect(screen.getAllByText(/far expiry is more expensive/i).length).toBeGreaterThan(0);
  });
});
