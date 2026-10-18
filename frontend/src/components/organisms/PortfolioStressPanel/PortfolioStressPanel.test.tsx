import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { replayPreset } from './helpers';
import { PortfolioStressPanel } from './PortfolioStressPanel';

describe('PortfolioStressPanel', () => {
  it('previews BTC and ETH shocks and shows the resulting book', async () => {
    const onPreview = vi.fn();
    renderWithProviders(
      <PortfolioStressPanel
        onPreview={onPreview}
        report={{
          equity: 10000,
          shockedEquity: 9895,
          equityDelta: -105,
          totalPnl: 0,
          shockedTotalPnl: -105,
          marginEquity: 20,
          shockedMarginEquity: 0,
          liquidated: 1,
        }}
      />,
    );
    expect(screen.getByTestId('portfolio-stress-result')).toHaveTextContent(/9[.,\s]?895/);
    await userEvent.click(screen.getByRole('button', { name: /preview|önizle/i }));
    expect(onPreview).toHaveBeenCalledWith({
      moves: [
        { asset: 'BTC', movePct: -10 },
        { asset: 'ETH', movePct: -15 },
      ],
    });
  });

  it('replays a chosen period and names a coin that stayed put', async () => {
    const onPreview = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <PortfolioStressPanel
        onPreview={onPreview}
        report={{
          equity: 10000,
          shockedEquity: 9800,
          equityDelta: -200,
          totalPnl: 0,
          shockedTotalPnl: -200,
          marginEquity: 0,
          shockedMarginEquity: 0,
          replay: {
            interval: '1d',
            moves: [
              {
                asset: 'BTC',
                movePct: -20,
                venues: [{ exchange: 'binance', symbol: 'BTCUSDT', movePct: -20, start: 100, end: 80 }],
              },
            ],
            missing: [{ asset: 'SOL', exchange: 'binance', symbol: 'SOLUSDT', reason: 'no_start' }],
          },
        }}
      />,
    );
    await user.click(screen.getByText(/replay a period|geçmişi tekrarla/i));
    await user.click(screen.getByRole('button', { name: /yesterday|dün/i }));
    await user.click(screen.getByRole('button', { name: /^replay$|^tekrarla$/i }));
    expect(onPreview).toHaveBeenCalledWith(replayPreset('yesterday'));
    const replay = screen.getByTestId('portfolio-stress-replay');
    expect(replay).toHaveTextContent(/BTC/);
    expect(replay).toHaveTextContent(/20/);
    expect(replay).toHaveTextContent(/SOLUSDT/);
    expect(screen.getByTestId('portfolio-stress-result')).toHaveTextContent(/9[.,\s]?800/);
  });
});
