import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { DetailChartToolbar } from './DetailChartToolbar';

describe('DetailChartToolbar', () => {
  it('renders refresh and calls onRefresh', async () => {
    const user = userEvent.setup();
    const onRefresh = vi.fn();
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h', '4h']}
        interval="1h"
        onIntervalChange={() => undefined}
        onRefresh={onRefresh}
      />,
    );
    await user.click(screen.getByRole('button', { name: /refresh/i }));
    expect(onRefresh).toHaveBeenCalled();
  });

  it('shows updating tag when isFetching', () => {
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        onRefresh={() => undefined}
        isFetching
      />,
    );
    expect(screen.getByText(/updating/i)).toBeInTheDocument();
  });

  it('exposes a single pump threshold select (not dual inputs)', () => {
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        pumpThresholdPct={5}
        onPumpThresholdChange={() => undefined}
        showPumpMarkers
        onShowPumpMarkersChange={() => undefined}
      />,
    );
    expect(screen.getByText(/pump threshold|pump eşiği/i)).toBeInTheDocument();
    // One switch for markers, no spinbutton number field
    expect(screen.queryByRole('spinbutton')).not.toBeInTheDocument();
    expect(screen.getByRole('switch')).toBeInTheDocument();
  });
});
