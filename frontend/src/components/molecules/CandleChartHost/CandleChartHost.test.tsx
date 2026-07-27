import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';

const remove = vi.fn();
const setData = vi.fn();
const applyOptions = vi.fn();
const fitContent = vi.fn();
const addSeries = vi.fn(() => ({
  setData,
  applyOptions,
  setMarkers: vi.fn(),
}));

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addSeries,
    applyOptions,
    remove,
    timeScale: () => ({ fitContent, applyOptions: vi.fn() }),
    priceScale: () => ({ applyOptions: vi.fn() }),
  })),
  CandlestickSeries: 'CandlestickSeries',
  LineSeries: 'LineSeries',
  CrosshairMode: { Normal: 0, Magnet: 1 },
  ColorType: { Solid: 0 },
}));

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
  });
});
