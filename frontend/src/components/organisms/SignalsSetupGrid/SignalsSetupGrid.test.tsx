import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { SwingSetup } from '@/libs/utils';
import { renderWithProviders } from '@/test/render';
import { SignalsSetupGrid } from './SignalsSetupGrid';

const setup: SwingSetup = {
  key: 'binance|BTCUSDT|4h',
  exchange: 'binance',
  symbol: 'BTCUSDT',
  interval: '4h',
  factors: ['ma_crossover', 'rsi'],
  score: 2,
  grade: 'B',
  sameBar: true,
  latestAt: '2026-08-01T08:00:00Z',
  summaries: ['RSI(14)=38.00 below 40'],
  hits: [],
};

describe('SignalsSetupGrid', () => {
  it('renders a setup card and opens the chart', async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    renderWithProviders(<SignalsSetupGrid setups={[setup]} onOpen={onOpen} />);
    expect(screen.getByText('BTC/USDT')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /open chart|grafiği aç/i }));
    expect(onOpen).toHaveBeenCalledWith('binance', 'BTCUSDT');
  });

  it('shows empty copy', () => {
    renderWithProviders(<SignalsSetupGrid setups={[]} emptyText="No setups" />);
    expect(screen.getByText('No setups')).toBeInTheDocument();
  });
});
