import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';

const remove = vi.fn();
const setData = vi.fn();
const applyOptions = vi.fn();
const fitContent = vi.fn();
const createPriceLine = vi.fn();
const addSeries = vi.fn(() => ({
  setData,
  applyOptions,
  createPriceLine,
}));

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addSeries,
    applyOptions,
    remove,
    timeScale: () => ({ fitContent }),
  })),
  LineSeries: {},
}));

// ResizeObserver polyfill for chart host
class RO {
  observe() {}
  unobserve() {}
  disconnect() {}
}
global.ResizeObserver = RO as unknown as typeof ResizeObserver;

import { IndicatorChartHost } from './IndicatorChartHost';

describe('IndicatorChartHost', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows skeleton when loading', () => {
    renderWithTheme(<IndicatorChartHost data={[]} isLoading />);
    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  it('renders host container when not loading', () => {
    renderWithTheme(
      <IndicatorChartHost data={[{ time: 1, value: 50 }, { time: 2, value: 55 }]} />,
    );
    expect(screen.getByTestId('indicator-chart-host')).toBeInTheDocument();
  });
});
