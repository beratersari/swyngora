import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { BasisCurveHistory } from './BasisCurveHistory';

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addSeries: () => ({ setData: vi.fn() }),
    timeScale: () => ({ fitContent: vi.fn() }),
    applyOptions: vi.fn(),
    remove: vi.fn(),
  })),
  LineSeries: {},
}));

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
vi.stubGlobal('ResizeObserver', ResizeObserverStub);

describe('BasisCurveHistory', () => {
  it('shows windows and a flip move', () => {
    renderWithProviders(
      <BasisCurveHistory
        venueLabel="Binance"
        curve={{
          complete: true,
          shape: 'contango',
          windows: [
            { window: '1h', complete: true, change: 'steeper', changePct: '+0.12' },
            { window: '4h', complete: true, change: 'flipped', pastShape: 'backwardation', shape: 'contango' },
            { window: '24h', complete: true, change: 'stable' },
            { window: '7d', complete: false },
          ],
          history: [
            { time: new Date(Date.now() - 3 * 3600_000).toISOString(), spreadPct: '-0.20' },
            { time: new Date(Date.now() - 2 * 3600_000).toISOString(), spreadPct: '0.10' },
            { time: new Date().toISOString(), spreadPct: '0.40' },
          ],
          moves: [
            {
              kind: 'flipped',
              fromShape: 'backwardation',
              toShape: 'contango',
              summary: 'Curve flipped from backwardation to contango.',
            },
          ],
        }}
      />,
    );
    expect(screen.getByTestId('basis-curve-history')).toBeInTheDocument();
    expect(screen.getByText(/over time/i)).toBeInTheDocument();
    expect(screen.getAllByText(/flipped/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/from backwardation to contango/i)).toBeInTheDocument();
  });
});
