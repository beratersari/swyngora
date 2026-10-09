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
let timeToIndexImpl: ((time: number, findNearest?: boolean) => number | null) | null = null;
const createPriceLine = vi.fn(() => ({ remove: vi.fn() }));
const removePriceLine = vi.fn();
const setStretchFactor = vi.fn();
const addSeries = vi.fn(() => ({
  setData,
  applyOptions,
  setMarkers: vi.fn(),
  attachPrimitive: vi.fn(),
  detachPrimitive: vi.fn(),
  createPriceLine,
  removePriceLine,
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
      ...(timeToIndexImpl ? { timeToIndex: timeToIndexImpl } : {}),
    }),
    priceScale: () => ({ applyOptions: vi.fn() }),
    panes: () => [{ setStretchFactor }, { setStretchFactor }],
    subscribeClick: vi.fn(),
    unsubscribeClick: vi.fn(),
  })),
  createSeriesMarkers: vi.fn(() => ({ setMarkers: vi.fn(), markers: () => [] })),
  CandlestickSeries: 'CandlestickSeries',
  LineSeries: 'LineSeries',
  HistogramSeries: 'HistogramSeries',
  LineStyle: { Solid: 0, Dashed: 2 },
  CrosshairMode: { Normal: 0, Magnet: 1 },
  ColorType: { Solid: 0 },
}));

import { createChart } from 'lightweight-charts';
import { CandleChartHost } from './CandleChartHost';

function seriesStub(extra?: Record<string, unknown>) {
  return {
    setData,
    applyOptions,
    setMarkers: vi.fn(),
    attachPrimitive: vi.fn(),
    detachPrimitive: vi.fn(),
    createPriceLine,
    removePriceLine,
    ...extra,
  };
}

describe('CandleChartHost', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    timeToIndexImpl = null;
    addSeries.mockImplementation(() => seriesStub());
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

  it('reports the candle time when the plot is clicked', () => {
    const onBarTimeClick = vi.fn();
    renderWithTheme(
      <CandleChartHost
        data={[{ time: 100, open: 1, high: 2, low: 0.5, close: 1.5 }]}
        onBarTimeClick={onBarTimeClick}
      />,
    );
    const chart = vi.mocked(createChart).mock.results.at(-1)?.value as {
      subscribeClick: ReturnType<typeof vi.fn>;
    };
    const handler = chart.subscribeClick.mock.calls[0]?.[0] as
      | ((param: { time?: number; point?: { x: number; y: number } }) => void)
      | undefined;
    handler?.({ time: 100, point: { x: 4, y: 8 } });
    handler?.({ time: 100 });
    expect(onBarTimeClick).toHaveBeenCalledTimes(1);
    expect(onBarTimeClick).toHaveBeenCalledWith(100);
  });

  it('snaps vertical lines onto existing candles without a whitespace series', () => {
    const attachPrimitive = vi.fn();
    addSeries.mockImplementation(() => seriesStub({ attachPrimitive }));
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
    // Desk overlay attaches on mount. The delist line attaches after it.
    expect(attachPrimitive).toHaveBeenCalledTimes(2);
  });

  it('adds a delta histogram pane and dashed liquidation prices', () => {
    renderWithTheme(
      <CandleChartHost
        data={[
          { time: 100, open: 1, high: 2, low: 0.5, close: 1.5 },
          { time: 200, open: 1.5, high: 2.2, low: 1.4, close: 2 },
        ]}
        lowerSeries={[
          {
            id: 'delta',
            pane: 1,
            kind: 'histogram',
            color: '#16C784',
            downColor: '#EA3943',
            title: 'Buy − sell',
            data: [
              { time: 100, value: 5 },
              { time: 200, value: -2 },
            ],
          },
        ]}
        priceLines={[{ id: 'liq-long', price: 90, color: '#EA3943', title: '100× Long' }]}
      />,
    );
    expect(addSeries).toHaveBeenCalledWith(
      'HistogramSeries',
      expect.objectContaining({ title: 'Buy − sell' }),
      1,
    );
    expect(setData).toHaveBeenCalledWith([
      { time: 100, value: 5, color: '#16C784' },
      { time: 200, value: -2, color: '#EA3943' },
    ]);
    expect(createPriceLine).toHaveBeenCalledWith(
      expect.objectContaining({ price: 90, title: '100× Long', lineStyle: 2 }),
    );
    expect(setStretchFactor).toHaveBeenCalled();
  });

  it('pins the latest candle by time-scale index when other series insert times', () => {
    timeToIndexImpl = () => 900;
    renderWithTheme(
      <CandleChartHost
        data={[
          { time: 100, open: 1, high: 2, low: 0.5, close: 1.5 },
          { time: 200, open: 1.5, high: 2.2, low: 1.4, close: 2 },
        ]}
        lowerSeries={[
          {
            id: 'open-liq-delta',
            pane: 1,
            kind: 'line',
            color: '#3861FB',
            title: 'Open long − short',
            data: [
              { time: 150, value: 10 },
              { time: 200, value: 25 },
            ],
          },
        ]}
      />,
    );
    // Index 900 is the last candle after earlier samples were inserted.
    // 80 visible bars and the default 6-slot right pad: from 821, to 907.
    expect(setVisibleLogicalRange).toHaveBeenCalledWith({ from: 821, to: 907 });
  });
});
