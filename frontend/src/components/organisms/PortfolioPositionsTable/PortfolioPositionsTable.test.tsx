import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { PortfolioPositionsTable } from './PortfolioPositionsTable';

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
    useGetPortfolioSituationChangeQuery: () => ({
      data: {
        liveOk: true,
        live: { symbol: 'BTCUSDT', priceCvd: { kind: 'agrees' } },
        comparison: {
          same: false,
          clauses: [{ clause: 'priceCvd', status: 'changed', entry: 'price_up_cvd_down', live: 'agrees' }],
        },
      },
      isLoading: false,
      isFetching: false,
    }),
  };
});

describe('PortfolioPositionsTable', () => {
  it('shows the entry read and the live read on a crypto position', () => {
    renderWithProviders(
      <PortfolioPositionsTable
        items={[
          {
            exchange: 'binance',
            symbol: 'BTCUSDT',
            quantity: 1,
            avgCost: 100,
            markPrice: 110,
            marketValue: 110,
            unrealizedPnL: 10,
            entrySituation: {
              symbol: 'BTCUSDT',
              priceCvd: { kind: 'price_up_cvd_down' },
            },
          },
          {
            exchange: 'nasdaq',
            symbol: 'AAPL',
            quantity: 1,
            avgCost: 100,
            markPrice: 100,
            marketValue: 100,
            unrealizedPnL: 0,
            entrySituation: {
              symbol: 'AAPL',
              priceCvd: { kind: 'agrees' },
            },
          },
        ]}
      />,
    );
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(screen.getByText('Now')).toBeInTheDocument();
    expect(screen.getByText('Price and futures CVD are moving together.')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Was: Price is rising while futures CVD is falling. Now: Price and futures CVD are moving together.',
      ),
    ).toBeInTheDocument();
    expect(screen.getAllByText('Entered as')).toHaveLength(1);
  });
});
