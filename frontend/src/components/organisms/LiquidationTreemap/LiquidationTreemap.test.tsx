import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { LiquidationTreemap } from './LiquidationTreemap';

describe('LiquidationTreemap', () => {
  it('renders square tiles and opens a coin', async () => {
    const onOpen = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <LiquidationTreemap
        onOpen={onOpen}
        coins={[
          { symbol: 'BTCUSDT', base: 'BTC', totalNotional: '1000', longNotional: '700', shortNotional: '300' },
          { symbol: 'ETHUSDT', base: 'ETH', totalNotional: '400', longNotional: '100', shortNotional: '300' },
        ]}
      />,
    );
    expect(screen.getByRole('group', { name: /liquidation coin map|likidasyon coin/i })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /BTCUSDT/i }));
    expect(onOpen).toHaveBeenCalledWith('BTCUSDT');
  });

  it('shows empty state', () => {
    renderWithProviders(<LiquidationTreemap coins={[]} />);
    expect(screen.getByText(/no liquidations|henüz likidasyon/i)).toBeInTheDocument();
  });
});
