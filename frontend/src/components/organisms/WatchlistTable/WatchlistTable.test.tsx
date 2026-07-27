import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { SpotMetricValue } from '@/components/molecules/SpotMetricValue';
import type { SpotMarket } from '@/libs/api';
import { WatchlistTable } from './WatchlistTable';

const ethSpot: SpotMarket = {
  symbol: 'ETHUSDT',
  lastPrice: '3200.5',
  priceChangePercent: '-0.75',
  quoteVolume: '500000000',
  marketCapCirculating: 3.8e11,
};

describe('WatchlistTable', () => {
  it('renders live price, change, volume, and mcap columns via renderMetric', async () => {
    const onOpen = vi.fn();
    const onRemove = vi.fn();
    renderWithProviders(
      <WatchlistTable
        items={[{ exchange: 'binance', symbol: 'ETHUSDT', addedAt: '2024-01-01T00:00:00Z' }]}
        loading={false}
        removeLoading={false}
        onOpen={onOpen}
        onRemove={onRemove}
        renderMetric={({ exchange, metric }) => (
          <SpotMetricValue metric={metric} spot={ethSpot} exchange={exchange} />
        )}
      />,
    );

    expect(screen.getByText('ETH/USDT')).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /last/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /24h/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /quote vol|quote hacim/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /mcap|circ|dolaşım/i })).toBeInTheDocument();

    expect(screen.getByText(/3[,.]?200/)).toBeInTheDocument();
    expect(screen.getByText(/-0\.75%/)).toBeInTheDocument();
    expect(screen.getByText(/500\.00M|500M/)).toBeInTheDocument();
    expect(screen.getByText(/380\.00B|380B/)).toBeInTheDocument();
  });

  it('calls onRemove without navigating when Remove is clicked', async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    const onRemove = vi.fn();
    renderWithProviders(
      <WatchlistTable
        items={[{ exchange: 'binance', symbol: 'ETHUSDT', addedAt: '2024-01-01T00:00:00Z' }]}
        loading={false}
        onOpen={onOpen}
        onRemove={onRemove}
      />,
    );

    await user.click(screen.getByRole('button', { name: /remove/i }));
    expect(onRemove).toHaveBeenCalledWith('binance', 'ETHUSDT');
    expect(onOpen).not.toHaveBeenCalled();
  });

  it('uses an icon-only remove control (no text label chrome)', () => {
    renderWithProviders(
      <WatchlistTable
        items={[{ exchange: 'binance', symbol: 'ETHUSDT', addedAt: '2024-01-01T00:00:00Z' }]}
        loading={false}
        onOpen={vi.fn()}
        onRemove={vi.fn()}
      />,
    );
    const btn = screen.getByRole('button', { name: /remove/i });
    expect(btn).toBeInTheDocument();
    // Label is aria-only — no visible "Remove" text node inside the button.
    expect(btn).toHaveTextContent('');
  });
});
