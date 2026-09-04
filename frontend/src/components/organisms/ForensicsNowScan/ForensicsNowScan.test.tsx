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
              whySimilar: 'Closest past twin matched most on open interest and the order book.',
              compared: [
                { name: 'oi', used: true, score: '91' },
                { name: 'book', used: true, score: '84' },
                { name: 'price', used: true, score: '72' },
                { name: 'volume', used: true, score: '41' },
                { name: 'takers', used: false },
              ],
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
    expect(screen.getByText(/open interest and the order book/i)).toBeInTheDocument();
    expect(screen.getByText(/open interest 91/i)).toBeInTheDocument();
    expect(screen.getByText(/taker flow n\/a/i)).toBeInTheDocument();
    await userEvent.click(screen.getByText('BTCUSDT'));
    expect(onOpen).toHaveBeenCalledWith('BTCUSDT');
    const alertLink = screen.getByRole('link', { name: /create alert|uyarı oluştur/i });
    expect(alertLink).toHaveAttribute('href', '/alerts?kind=setup_radar&symbol=BTCUSDT');
  });
});
