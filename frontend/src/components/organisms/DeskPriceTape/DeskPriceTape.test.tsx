import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { DeskPriceTape } from './DeskPriceTape';

describe('DeskPriceTape', () => {
  it('switches source and renders tape links', async () => {
    const user = userEvent.setup();
    const onSourceChange = vi.fn();
    renderWithProviders(
      <DeskPriceTape
        source="binance"
        onSourceChange={onSourceChange}
        sourceAriaLabel="Price tape source"
        tapeAriaLabel="Live market prices"
        items={[
          {
            exchange: 'binance',
            symbol: 'BTCUSDT',
            lastPrice: '67,000',
            changePercent: '+1.20%',
            changeValue: 1.2,
            href: '/markets/binance/BTCUSDT',
          },
        ]}
      />,
    );

    expect(screen.getByRole('tab', { name: /binance/i })).toHaveAttribute('aria-selected', 'true');
    const links = screen.getAllByRole('link', { name: /BTC\/USDT/i });
    expect(links[0]).toHaveAttribute('href', '/markets/binance/BTCUSDT');

    await user.click(screen.getByRole('tab', { name: /watchlist|izleme/i }));
    expect(onSourceChange).toHaveBeenCalledWith('watchlist');
  });

  it('shows an empty watchlist hint', () => {
    renderWithProviders(
      <DeskPriceTape
        source="watchlist"
        onSourceChange={() => undefined}
        sourceAriaLabel="Price tape source"
        tapeAriaLabel="Live market prices"
        items={[]}
        emptyLabel="Add symbols to your watchlist"
      />,
    );
    expect(screen.getByText('Add symbols to your watchlist')).toBeInTheDocument();
  });
});
