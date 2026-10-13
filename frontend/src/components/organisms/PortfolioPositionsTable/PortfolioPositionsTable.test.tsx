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
          clauses: [
            { clause: 'priceCvd', status: 'changed', entry: 'price_up_cvd_down', live: 'agrees' },
          ],
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

  it('shows an earlier buy beside the newest one', () => {
    renderWithProviders(
      <PortfolioPositionsTable
        items={[
          {
            exchange: 'binance',
            symbol: 'BTCUSDT',
            quantity: 1.4,
            avgCost: 100,
            markPrice: 110,
            marketValue: 154,
            unrealizedPnL: 14,
            openReads: [
              {
                id: 'new',
                quantity: 0.4,
                openedAt: '2026-10-12T12:00:00Z',
                newest: true,
                situation: { symbol: 'BTCUSDT', priceCvd: { kind: 'price_down_cvd_up' } },
              },
              {
                id: 'old',
                quantity: 1,
                openedAt: '2026-10-12T10:00:00Z',
                situation: { symbol: 'BTCUSDT', priceCvd: { kind: 'price_up_cvd_down' } },
              },
            ],
          },
        ]}
      />,
    );
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('Earlier buy')).toBeInTheDocument();
    expect(screen.getByText('The situation alert follows this buy.')).toBeInTheDocument();
    expect(screen.getByText('Price is falling while futures CVD is rising.')).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
  });
});
