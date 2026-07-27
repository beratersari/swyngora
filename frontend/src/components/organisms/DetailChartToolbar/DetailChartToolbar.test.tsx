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
        limit={100}
        onIntervalChange={() => undefined}
        onLimitChange={() => undefined}
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
        limit={100}
        onIntervalChange={() => undefined}
        onLimitChange={() => undefined}
        onRefresh={() => undefined}
        isFetching
      />,
    );
    expect(screen.getByText(/updating/i)).toBeInTheDocument();
  });
});
