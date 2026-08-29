import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { IndicatorPanel } from './IndicatorPanel';

vi.mock('@/components/molecules/IndicatorChartHost', () => ({
  IndicatorChartHost: () => <div data-testid="rsi-chart" />,
}));

describe('IndicatorPanel', () => {
  it('shows error alert', () => {
    renderWithTheme(<IndicatorPanel errorMessage="boom" />);
    expect(screen.getByText('boom')).toBeInTheDocument();
  });

  it('formats latest EMA in the display currency when priceQuote is set', () => {
    renderWithTheme(
      <IndicatorPanel
        priceQuote="TRY"
        data={{
          latest: { rsi: 55, zone: 'neutral', ema: { '12': 400 } },
        }}
      />,
    );
    expect(screen.getByText(/400/)).toBeInTheDocument();
    expect(screen.getByText(/TRY/)).toBeInTheDocument();
  });

  it('renders RSI snapshot and chart when data present', () => {
    renderWithTheme(
      <IndicatorPanel
        data={{
          rsiPeriod: 14,
          latest: { rsi: 55, zone: 'neutral', ema: { '12': 100, '26': 99 } },
          points: [{ openTime: '2024-01-01T00:00:00Z', rsi: 55, ema: { '12': 100 } }],
        }}
      />,
    );
    expect(screen.getByTestId('rsi-chart')).toBeInTheDocument();
    expect(screen.getByText(/55/)).toBeInTheDocument();
  });

  it('toggles EMA switch when handler provided', async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    renderWithTheme(
      <IndicatorPanel
        data={{ latest: { rsi: 50, zone: 'neutral', ema: { '12': 1 } } }}
        showEmaOnChart
        onToggleEma={onToggle}
      />,
    );
    const sw = screen.getByRole('switch');
    await user.click(sw);
    expect(onToggle).toHaveBeenCalled();
  });
});
