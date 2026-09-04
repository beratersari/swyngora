import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { ForensicsNowScan } from './ForensicsNowScan';

describe('ForensicsNowScan', () => {
  it('renders ranked coins and after stories', async () => {
    const onOpen = vi.fn();
    renderWithProviders(
      <ForensicsNowScan
        scan={{
          summary: '1 of 4 scanned coin(s) currently look like a past pre-move setup.',
          hits: [
            {
              symbol: 'BTCUSDT',
              similarity: '82',
              lean: 'up',
              afterStory: 'When BTC looked like this before, price more often kept rising over the next 1h.',
              afterHorizons: [{ horizon: '1h', up: 3, down: 0, medianPct: '+2.1' }],
            },
          ],
        }}
        onOpenSymbol={onOpen}
      />,
    );
    expect(screen.getByText(/1 of 4 scanned/i)).toBeInTheDocument();
    expect(screen.getByText('BTCUSDT')).toBeInTheDocument();
    expect(screen.getByText(/kept rising/i)).toBeInTheDocument();
    await userEvent.click(screen.getByText('BTCUSDT'));
    expect(onOpen).toHaveBeenCalledWith('BTCUSDT');
  });
});
