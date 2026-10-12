import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { i18n } from '@/libs/i18n';
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
    expect(screen.getAllByText('Entered as')).toHaveLength(1);
  });

  it('shows the price move after the fill on an equity position', async () => {
    await i18n.changeLanguage('en');
    renderWithProviders(
      <PortfolioPositionsTable
        items={[
          {
            exchange: 'nasdaq',
            symbol: 'AAPL',
            quantity: 1,
            avgCost: 150,
            markPrice: 180,
            moves: [
              { window: '1h', status: 'ready', pct: 2, price: 102 },
              { window: '4h', status: 'pending' },
              { window: '24h', status: 'missing' },
            ],
          },
        ]}
      />,
    );
    expect(screen.getByText('1h +2.00%')).toBeInTheDocument();
    expect(screen.getByText('4h pending')).toBeInTheDocument();
    expect(screen.getByText('24h —')).toBeInTheDocument();
    expect(screen.queryByText('Entered as')).not.toBeInTheDocument();
    expect(screen.queryByText('24h 0.00%')).not.toBeInTheDocument();
  });
});
