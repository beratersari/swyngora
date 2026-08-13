import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { PriceChangeHeatmap } from './PriceChangeHeatmap';

describe('PriceChangeHeatmap', () => {
  it('renders tiles and opens a market on click', () => {
    const onOpen = vi.fn();
    renderWithTheme(
      <PriceChangeHeatmap
        onOpen={onOpen}
        items={[
          {
            symbol: 'BTCUSDT',
            exchange: 'binance',
            lastPrice: '100',
            priceChangePercent: '2.5',
            quoteVolume: '9000',
          },
          {
            symbol: 'ETHUSDT',
            exchange: 'binance',
            lastPrice: '10',
            priceChangePercent: '-1.2',
            quoteVolume: '3000',
          },
        ]}
      />,
    );
    expect(screen.getByRole('group', { name: /heatmap|ısı/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /BTC/i }));
    expect(onOpen).toHaveBeenCalledWith('binance', 'BTCUSDT');
  });

  it('shows empty state', () => {
    renderWithTheme(<PriceChangeHeatmap items={[]} />);
    expect(screen.getByText(/no markets|piyasa yok/i)).toBeInTheDocument();
  });
});
