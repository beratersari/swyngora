import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { PortfolioMarginPositionsTable } from './PortfolioMarginPositionsTable';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketSituationQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        priceCvd: { kind: 'agrees' },
      },
      isLoading: false,
      isFetching: false,
    }),
  };
});

describe('PortfolioMarginPositionsTable', () => {
  it('shows the placement read and the live read on an open margin position', () => {
    renderWithProviders(
      <PortfolioMarginPositionsTable
        items={[
          {
            id: 'm1',
            exchange: 'binance',
            symbol: 'BTCUSDT',
            side: 'long',
            leverage: 5,
            quantity: 1,
            status: 'open',
            entrySituation: {
              symbol: 'BTCUSDT',
              priceCvd: { kind: 'price_up_cvd_down' },
            },
          },
          {
            id: 'eq',
            exchange: 'nasdaq',
            symbol: 'AAPL',
            side: 'long',
            leverage: 2,
            quantity: 1,
            status: 'open',
            entrySituation: {
              symbol: 'AAPL',
              priceCvd: { kind: 'agrees' },
            },
          },
        ]}
      />,
    );
    expect(screen.getByText('Margin long 5x')).toBeInTheDocument();
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(screen.getByText('Now')).toBeInTheDocument();
    expect(screen.getByText('Price and futures CVD are moving together.')).toBeInTheDocument();
    expect(screen.queryByText('Margin long 2x')).not.toBeInTheDocument();
  });
});
