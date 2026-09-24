import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
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
    expect(screen.getByTestId('portfolio-stress-result')).toHaveTextContent(/9,?895/);
    await userEvent.click(screen.getByRole('button', { name: /preview|önizle/i }));
    expect(onPreview).toHaveBeenCalledWith([
      { asset: 'BTC', movePct: -10 },
      { asset: 'ETH', movePct: -15 },
    ]);
  });
});
