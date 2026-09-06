import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OptionsHistory } from './OptionsHistory';

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

describe('OptionsHistory', () => {
  it('shows window chips when history is present', () => {
    renderWithProviders(
      <OptionsHistory
        data={{
          history: {
            windows: [
              { window: '24h', complete: true, oi: { changePct: '8.5', complete: true } },
            ],
            points: [
              { time: '2026-03-02T00:00:00Z', openInterestUsd: '100', dvol: '49' },
              { time: new Date().toISOString(), openInterestUsd: '108', dvol: '51' },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText('+8.5%')).toBeInTheDocument();
    expect(screen.getByLabelText(/series/i)).toBeInTheDocument();
  });

  it('shows an empty hint when there is no series yet', () => {
    renderWithProviders(<OptionsHistory data={{}} />);
    expect(screen.getByText(/collecting|no history|over time/i)).toBeInTheDocument();
  });
});
