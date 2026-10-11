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
import { evenBlendVenueIds, formatLeverageMix, pickGrid } from './LiquidationHeatmap.helpers';
import type { LiquidationHeatmapData } from './LiquidationHeatmap.types';

const data: LiquidationHeatmapData = {
  symbol: 'BTCUSDT',
  range: '24h',
  leverageMix: [
    { leverage: 5, weight: 0.15 },
    { leverage: 10, weight: 0.25 },
  ],
  missingVenues: ['bybit'],
  okx: {
    exchange: 'okx',
    accountBlend: 'even',
    totals: [
      [1, 0, 0],
      [0, 2, 0],
    ],
    longs: [
      [1, 0, 0],
      [0, 0, 0],
    ],
    shorts: [
      [0, 0, 0],
      [0, 2, 0],
    ],
    maxIntensity: 2,
    coverage: 1,
    columnsWithOi: 2,
  },
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
          signals: 12,
          validated: 10,
          missing: 2,
          priceMissing: 1,
          liqMissing: 1,
          hits: 6,
          falseSignals: 4,
          hitRate: 0.6,
          avgTimeToHitSec: 1800,
          liqIncreased: 4,
          liqIncreaseRate: 0.67,
          pending: 0,
        },
      ],
      signals: [
        {
          time: '2026-08-29T12:00:00Z',
          priceLo: 108_000,
          priceHi: 110_000,
          side: 'short',
          horizons: [
            { horizon: '1h', status: 'hit', validated: true, timeToHitSec: 1800, horizonSec: 3600 },
            { horizon: '4h', status: 'hit', validated: true, timeToHitSec: 1800, horizonSec: 14400 },
            { horizon: '12h', status: 'price_gap', priceCoveredSec: 3600, horizonSec: 43200 },
          ],
        },
      ],
    },
  },
};

describe('liquidation heatmap helpers', () => {
  it('keeps OKX off the combined grid and formats the leverage mix', () => {
    expect(pickGrid(data, 'okx')?.exchange).toBe('okx');
    expect(pickGrid(data, 'combined')?.exchange).toBe('combined');
    expect(formatLeverageMix(data.leverageMix)).toBe('5× 15% · 10× 25%');
    expect(evenBlendVenueIds(data, 'combined')).toEqual(['okx']);
    expect(formatLeverageMix(undefined)).toBe('');
  });
});

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
    expect(screen.getByText('Validated')).toBeInTheDocument();
    expect(screen.getByText('60%')).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-heatmap-signals')).toBeInTheDocument();
    expect(screen.getAllByText('Hit · 30m').length).toBeGreaterThan(0);
    expect(screen.getByText('Price gap · 1h/12h')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '12h' }));
    expect(onRangeChange).toHaveBeenCalledWith('12h');
    await user.click(screen.getByRole('button', { name: '90d' }));
    expect(onRangeChange).toHaveBeenCalledWith('90d');
    expect(screen.getByTestId('liquidation-heatmap-mix')).toHaveTextContent('5× 15%');
    expect(screen.getByTestId('liquidation-heatmap-mix')).toHaveTextContent('10× 25%');
    expect(screen.getByTestId('liquidation-heatmap-even')).toHaveTextContent('OKX');
    expect(screen.getByTestId('liquidation-heatmap-missing')).toHaveTextContent('Bybit');
    await user.click(screen.getByRole('button', { name: 'Binance' }));
    expect(onVenueChange).toHaveBeenCalledWith('binance');
    await user.click(screen.getByRole('button', { name: 'OKX' }));
    expect(onVenueChange).toHaveBeenCalledWith('okx');
    await user.click(screen.getByRole('button', { name: 'Longs' }));
    expect(onSideChange).toHaveBeenCalledWith('longs');
  });

  it('says a model range past a day reads the stored archive', () => {
    const props = {
      data,
      onRangeChange: () => undefined,
      venue: 'combined' as const,
      onVenueChange: () => undefined,
      side: 'totals' as const,
      onSideChange: () => undefined,
      source: 'model' as const,
    };
    const { rerender } = renderWithProviders(<LiquidationHeatmap {...props} range="30d" />);
    expect(screen.getByTestId('liquidation-heatmap-archive')).toHaveTextContent(/stored open-interest sample/i);
    rerender(<LiquidationHeatmap {...props} range="24h" />);
    expect(screen.queryByTestId('liquidation-heatmap-archive')).not.toBeInTheDocument();
    rerender(<LiquidationHeatmap {...props} range="30d" source="prints" />);
    expect(screen.queryByTestId('liquidation-heatmap-archive')).not.toBeInTheDocument();
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
    expect(screen.queryByRole('button', { name: 'Prints' })).not.toBeInTheDocument();
  });

  it('shows measured prints and hides the model review', async () => {
    const onSourceChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <LiquidationHeatmap
        data={{ ...data, kind: 'prints', leverageMix: [] }}
        range="24h"
        onRangeChange={() => undefined}
        venue="combined"
        onVenueChange={() => undefined}
        side="totals"
        onSideChange={() => undefined}
        source="prints"
        onSourceChange={onSourceChange}
      />,
    );
    expect(screen.getByText(/Stored liquidation prints/i)).toBeInTheDocument();
    expect(screen.queryByTestId('liquidation-heatmap-gap')).not.toBeInTheDocument();
    expect(screen.getByTestId('liquidation-heatmap-missing')).toHaveTextContent(/was not recorded/i);
    expect(screen.getByTestId('liquidation-heatmap-coverage')).toHaveTextContent(/were recorded/i);
    expect(screen.queryByTestId('liquidation-heatmap-review')).not.toBeInTheDocument();
    expect(screen.queryByTestId('liquidation-heatmap-mix')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Model' }));
    expect(onSourceChange).toHaveBeenCalledWith('model');
  });

  it('names an unrecorded print column and leaves a quiet recorded hour unmarked', () => {
    const props = {
      data: {
        ...data,
        kind: 'prints' as const,
        leverageMix: [],
        binance: { exchange: 'binance', hasData: [true, false] },
        bybit: { exchange: 'bybit', hasData: [true, true] },
        okx: { exchange: 'okx', hasData: [false, true] },
        combined: { ...data.combined, hasData: [false, false] },
      },
      range: '24h' as const,
      onRangeChange: () => undefined,
      venue: 'combined' as const,
      onVenueChange: () => undefined,
      side: 'totals' as const,
      onSideChange: () => undefined,
      source: 'prints' as const,
    };
    const { rerender } = renderWithProviders(<LiquidationHeatmap {...props} />);
    const gap = screen.getByTestId('liquidation-heatmap-gap');
    expect(gap).toHaveTextContent('Binance');
    expect(gap).toHaveTextContent('OKX');
    expect(gap).not.toHaveTextContent('Bybit');
    expect(gap).toHaveTextContent(/quiet recorded hour stays empty/i);
    rerender(<LiquidationHeatmap {...props} source="model" />);
    expect(screen.queryByTestId('liquidation-heatmap-gap')).not.toBeInTheDocument();
  });
});
