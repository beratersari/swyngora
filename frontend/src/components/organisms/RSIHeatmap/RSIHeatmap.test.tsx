import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { RSIHeatmap } from './RSIHeatmap';

describe('RSIHeatmap', () => {
  it('plots each pair as a labeled dot and opens it on click', async () => {
    const onOpen = vi.fn();
    renderWithProviders(
      <RSIHeatmap
        data={{
          exchange: 'binance',
          interval: '1h',
          averageRsi: 50,
          oversoldCount: 1,
          overboughtCount: 0,
          items: [
            { rank: 1, symbol: 'BTCUSDT', base: 'BTC', rsi: 28, zone: 'oversold', marketCapCirculating: 1_000_000 },
          ],
        }}
        onOpen={onOpen}
      />,
    );
    expect(screen.getByTestId('rsi-heatmap')).toBeInTheDocument();
    expect(screen.getByTestId('rsi-avg-line')).toBeInTheDocument();
    expect(screen.getByText('BTC')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('rsi-dot-BTC'));
    expect(onOpen).toHaveBeenCalledWith('binance', 'BTCUSDT');
  });
});
