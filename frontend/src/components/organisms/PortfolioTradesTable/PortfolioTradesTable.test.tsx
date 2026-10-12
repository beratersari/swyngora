import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { PortfolioTradesTable } from './PortfolioTradesTable';

describe('PortfolioTradesTable', () => {
  it('names the fill as the moment the stored situation was read', () => {
    renderWithProviders(
      <PortfolioTradesTable
        items={[
          {
            id: 't1',
            exchange: 'binance',
            symbol: 'BTCUSDT',
            side: 'buy',
            quantity: 1,
            price: 100,
            situation: { symbol: 'BTCUSDT', priceCvd: { kind: 'agrees' } },
          },
        ]}
      />,
    );
    expect(screen.getByText('Market when this order filled.')).toBeInTheDocument();
    expect(screen.getByText('Price and futures CVD are moving together.')).toBeInTheDocument();
  });

  it('names what left between the buy and the sell', () => {
    renderWithProviders(
      <PortfolioTradesTable
        items={[
          {
            id: 'sell-1',
            exchange: 'binance',
            symbol: 'BTCUSDT',
            side: 'sell',
            quantity: 1,
            price: 110,
            situation: { symbol: 'BTCUSDT', priceCvd: { kind: 'agrees' } },
            left: [
              {
                sourceTradeId: 'buy-1',
                entry: { symbol: 'BTCUSDT', priceCvd: { kind: 'price_up_cvd_down' } },
                comparison: {
                  same: false,
                  clauses: [
                    { clause: 'priceCvd', status: 'changed', entry: 'price_up_cvd_down', live: 'agrees' },
                  ],
                },
              },
            ],
          },
        ]}
      />,
    );
    expect(
      screen.getByText(
        'Was: Price is rising while futures CVD is falling. Now: Price and futures CVD are moving together.',
      ),
    ).toBeInTheDocument();
  });
});
