import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';

const remove = vi.fn();
const setData = vi.fn();
const applyOptions = vi.fn();
const fitContent = vi.fn();
const subscribeVisibleLogicalRangeChange = vi.fn();
const unsubscribeVisibleLogicalRangeChange = vi.fn();
const getVisibleLogicalRange = vi.fn(() => null);
const setVisibleLogicalRange = vi.fn();
const addSeries = vi.fn(() => ({
  setData,
  applyOptions,
  setMarkers: vi.fn(),
  attachPrimitive: vi.fn(),
  detachPrimitive: vi.fn(),
}));

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addSeries,
    applyOptions,
    remove,
    timeScale: () => ({
      fitContent,
      applyOptions: vi.fn(),
      subscribeVisibleLogicalRangeChange,
      unsubscribeVisibleLogicalRangeChange,
      getVisibleLogicalRange,
      setVisibleLogicalRange,
    }),
    priceScale: () => ({ applyOptions: vi.fn() }),
  })),
  createSeriesMarkers: vi.fn(() => ({ setMarkers: vi.fn(), markers: () => [] })),
  CandlestickSeries: 'CandlestickSeries',
  LineSeries: 'LineSeries',
  CrosshairMode: { Normal: 0, Magnet: 1 },
  ColorType: { Solid: 0 },
}));

import { createChart } from 'lightweight-charts';
import { CandleChartHost } from './CandleChartHost';

describe('CandleChartHost', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows skeleton when loading with no data', () => {
    renderWithTheme(<CandleChartHost data={[]} isLoading />);
    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  it('mounts chart host when data present', () => {
    renderWithTheme(
      <CandleChartHost
        data={[{ time: 1, open: 1, high: 2, low: 0.5, close: 1.5 }]}
      />,
    );
    expect(screen.getByTestId('candle-chart-host')).toBeInTheDocument();
    expect(subscribeVisibleLogicalRangeChange).toHaveBeenCalled();
    expect(createChart).toHaveBeenCalled();
    const opts = vi.mocked(createChart).mock.calls[0]?.[1] as {
      timeScale?: { timeVisible?: boolean; secondsVisible?: boolean };
    };
    expect(opts.timeScale?.timeVisible).toBe(true);
    expect(opts.timeScale?.secondsVisible).toBe(true);
  });

  it('snaps vertical lines onto existing candles without a whitespace series', () => {
    const attachPrimitive = vi.fn();
    addSeries.mockImplementation(() => ({
      setData,
      applyOptions,
      setMarkers: vi.fn(),
      attachPrimitive,
      detachPrimitive: vi.fn(),
    }));
    renderWithTheme(
      <CandleChartHost
        data={[
          { time: 100, open: 1, high: 2, low: 0.5, close: 1.5 },
          { time: 200, open: 1.5, high: 2.2, low: 1.4, close: 2 },
        ]}
        vertLines={[{ id: 'delist-halt', time: 250, color: '#f00', label: 'Delist' }]}
      />,
    );
    expect(addSeries).toHaveBeenCalledTimes(1);
    expect(attachPrimitive).toHaveBeenCalledTimes(1);
  });
});
