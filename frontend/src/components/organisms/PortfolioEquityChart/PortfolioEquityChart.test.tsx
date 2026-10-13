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
    expect(screen.getByText(/Cash you add or take out is not part of this line/)).toBeInTheDocument();
  });

  it('shows period trading P&L', () => {
    renderWithTheme(
      <PortfolioEquityChart
        points={[
          { t: '2026-08-01T10:00:00.000Z', equity: 10000, externalFlow: 0 },
          { t: '2026-08-01T15:00:00.000Z', equity: 16000, externalFlow: 5000 },
        ]}
        period="1d"
        onPeriodChange={() => {}}
        changeAmount={20}
        changePct={0.2}
        currency="USDT"
      />,
    );
    expect(screen.getByText('Period P&L +20 USDT (+0.2%)')).toBeInTheDocument();
  });
});
