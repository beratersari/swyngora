import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { EmergencyStop } from './EmergencyStop';

const save = vi.fn();

vi.mock('@/libs/api', () => ({
  useGetEmergencyStopQuery: () => ({
    data: {
      enabled: true,
      lossPct: 5,
      dailyPnlPct: -8.2,
      hit: true,
      tripped: true,
      run: {
        actions: [
          { kind: 'close_spot', targetId: 'binance:BTCUSDT', symbol: 'BTCUSDT', status: 'ok' },
          {
            kind: 'close_margin',
            targetId: 'm1',
            symbol: 'ETHUSDT',
            status: 'failed',
            detail: 'book cannot hold this size',
          },
        ],
      },
    },
  }),
  useSetEmergencyStopMutation: () => [save, { isLoading: false }],
}));

describe('EmergencyStop', () => {
  it('shows which steps worked and saves the switch', async () => {
    const user = userEvent.setup();
    renderWithTheme(<EmergencyStop portfolioId="b1" />);
    expect(screen.getByText('Today -8.20%')).toBeInTheDocument();
    expect(screen.getByText(/Close BTCUSDT/)).toBeInTheDocument();
    expect(screen.getByText(/worked/)).toBeInTheDocument();
    expect(screen.getByText(/Close margin ETHUSDT/)).toBeInTheDocument();
    expect(screen.getByText(/book cannot hold this size/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(save).toHaveBeenCalledWith({ portfolioId: 'b1', enabled: true, lossPct: 5 });
  });
});
