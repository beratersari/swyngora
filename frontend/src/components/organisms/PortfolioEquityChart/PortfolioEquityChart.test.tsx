import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { PortfolioEquityChart } from './PortfolioEquityChart';

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addSeries: () => ({ setData: vi.fn() }),
    timeScale: () => ({ fitContent: vi.fn() }),
    applyOptions: vi.fn(),
    remove: vi.fn(),
  })),
  LineSeries: {},
  TickMarkType: { DayOfMonth: 2, Month: 1, Time: 3, Year: 0 },
}));

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
vi.stubGlobal('ResizeObserver', ResizeObserverStub);

describe('PortfolioEquityChart', () => {
  it('shows the max drawdown under the equity title', () => {
    renderWithTheme(
      <PortfolioEquityChart
        points={[
          { t: '2026-08-01T10:00:00.000Z', equity: 10000 },
          { t: '2026-08-01T15:00:00.000Z', equity: 8000 },
        ]}
        period="1d"
        onPeriodChange={() => {}}
        maxDrawdown={2000}
        maxDrawdownPct={20}
        drawdownFrom="2026-08-01T10:00:00.000Z"
        drawdownTo="2026-08-01T15:00:00.000Z"
      />,
    );
    expect(screen.getByText(/Max drawdown 2.00K \(20%\)/)).toBeInTheDocument();
  });
});
