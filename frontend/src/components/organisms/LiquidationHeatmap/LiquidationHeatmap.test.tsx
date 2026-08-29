import { beforeAll, describe, expect, it, vi } from 'vitest';

beforeAll(() => {
  Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
    value: () => null,
  });
});
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { LiquidationHeatmap } from './LiquidationHeatmap';
import type { LiquidationHeatmapData } from './LiquidationHeatmap.types';

const data: LiquidationHeatmapData = {
  symbol: 'BTCUSDT',
  range: '24h',
  prices: [110_000, 100_000, 90_000],
  times: ['2026-08-29T12:00:00Z', '2026-08-29T12:30:00Z'],
  combined: {
    exchange: 'combined',
    totals: [
      [1, 2, 3],
      [4, 5, 6],
    ],
    longs: [
      [1, 0, 3],
      [0, 5, 0],
    ],
    shorts: [
      [0, 2, 0],
      [4, 0, 6],
    ],
    maxIntensity: 6,
    coverage: 1,
    columnsWithOi: 2,
  },
  review: {
    combined: {
      exchange: 'combined',
      horizons: [
        {
          horizon: '1h',
          signals: 10,
          hits: 6,
          falseSignals: 4,
          hitRate: 0.6,
          avgTimeToHitSec: 1800,
          liqIncreased: 4,
          liqIncreaseRate: 0.67,
          pending: 2,
        },
      ],
    },
  },
};

describe('LiquidationHeatmap', () => {
  it('renders range, venue, and side chips', async () => {
    const onRangeChange = vi.fn();
    const onVenueChange = vi.fn();
    const onSideChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <LiquidationHeatmap
        data={data}
        range="24h"
        onRangeChange={onRangeChange}
        venue="combined"
        onVenueChange={onVenueChange}
        side="totals"
        onSideChange={onSideChange}
      />,
    );
    expect(screen.getByTestId('liquidation-heatmap')).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-heatmap-canvas')).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-heatmap-review')).toBeInTheDocument();
    expect(screen.getByText('60%')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '12h' }));
    expect(onRangeChange).toHaveBeenCalledWith('12h');
    await user.click(screen.getByRole('button', { name: 'Binance' }));
    expect(onVenueChange).toHaveBeenCalledWith('binance');
    await user.click(screen.getByRole('button', { name: 'Longs' }));
    expect(onSideChange).toHaveBeenCalledWith('longs');
  });

  it('shows empty copy when there is no grid yet', () => {
    renderWithProviders(
      <LiquidationHeatmap
        data={{}}
        range="24h"
        onRangeChange={() => undefined}
        venue="combined"
        onVenueChange={() => undefined}
        side="totals"
        onSideChange={() => undefined}
      />,
    );
    expect(screen.getByText(/No liquidation map/i)).toBeInTheDocument();
  });
});
