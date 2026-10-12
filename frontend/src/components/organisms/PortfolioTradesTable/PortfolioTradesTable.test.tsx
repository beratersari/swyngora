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
});
