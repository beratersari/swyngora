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
          lastPrice: '100',
          levels: [
            { price: '110', longNotional: '10', shortNotional: '40', totalNotional: '50' },
            { price: '90', longNotional: '30', shortNotional: '5', totalNotional: '35' },
          ],
        }}
      />,
    );
    expect(screen.getByTestId('liquidation-bar-chart')).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-bar-chart-canvas')).toBeInTheDocument();
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
