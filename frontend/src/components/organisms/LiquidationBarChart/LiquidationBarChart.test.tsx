import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { LiquidationBarChart } from './LiquidationBarChart';

describe('LiquidationBarChart', () => {
  it('renders price-level bars', () => {
    renderWithProviders(
      <LiquidationBarChart
        data={{
          kind: 'levels',
          symbol: 'BTCUSDT',
          exchange: 'all',
          lastPrice: '100',
          levels: [
            { price: '110', longNotional: '10', shortNotional: '40', totalNotional: '50' },
            { price: '90', longNotional: '30', shortNotional: '5', totalNotional: '35' },
          ],
          review: {
            summary: 'Of 4 estimated levels with later tape, 2 printed nearby in 1h (50%).',
            combined: {
              horizons: [{ horizon: '1h', signals: 8, validated: 4, pending: 2, hits: 2, hitRate: 0.5, fillRate: 0.2 }],
              byLeverage: [
                { leverage: 100, side: 'short', validated: 4, hitRate: 0.6, fillRate: 0.2, reliability: 'high' },
              ],
              samples: [
                {
                  time: '2026-09-01T12:00:00.000Z',
                  lastPrice: '100',
                  bands: [
                    { leverage: 100, side: 'short', predicted: '40000', realized: '8000', status: 'hit' },
                  ],
                },
              ],
            },
          },
        }}
      />,
    );
    expect(screen.getByTestId('liquidation-bar-chart')).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-bar-chart-canvas')).toBeInTheDocument();
    expect(screen.getByText(/^From last$|^Last’ten$/)).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-levels-review')).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-levels-reliability')).toBeInTheDocument();
  });

  it('renders market-wide time bars', () => {
    renderWithProviders(
      <LiquidationBarChart
        data={{
          kind: 'totals',
          symbol: 'all',
          bars: [
            { t: '2026-08-30T12:00:00.000Z', longNotional: '20', shortNotional: '10', totalNotional: '30' },
          ],
        }}
      />,
    );
    expect(screen.getByText(/total liquidations|toplam likidasyon/i)).toBeInTheDocument();
  });
});
