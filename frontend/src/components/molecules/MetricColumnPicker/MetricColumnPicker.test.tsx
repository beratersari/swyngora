import { describe, expect, it, vi } from 'vitest';
import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { metricsForSurface } from '@/libs/utils/spotMetrics';
import { MetricColumnPicker } from './MetricColumnPicker';

describe('MetricColumnPicker', () => {
  it('opens panel and can move a metric down', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const available = metricsForSurface('markets');
    renderWithTheme(
      <MetricColumnPicker
        available={available}
        value={['lastPrice', 'quoteVolume']}
        onChange={onChange}
        getLabel={(k) => k}
        ariaLabel="Columns"
        buttonLabel="Columns"
        moveUpLabel="Move up"
        moveDownLabel="Move down"
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Columns' }));
    const list = await screen.findByRole('list', { name: 'Columns' });
    expect(within(list).getByText('last')).toBeInTheDocument();
    expect(within(list).getByText('quoteVol')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Move down: last/i }));
    expect(onChange).toHaveBeenCalledWith(['quoteVolume', 'lastPrice']);
  });

  it('adds an unselected metric to the end', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderWithTheme(
      <MetricColumnPicker
        available={metricsForSurface('watchlist')}
        value={['lastPrice']}
        onChange={onChange}
        getLabel={(k) => k}
        ariaLabel="Columns"
        buttonLabel="Columns"
      />,
    );
    await user.click(screen.getByRole('button', { name: 'Columns' }));
    const highLabel = await screen.findByText('high');
    const row = highLabel.closest('label');
    expect(row).toBeTruthy();
    await user.click(within(row as HTMLElement).getByRole('checkbox'));
    expect(onChange).toHaveBeenCalledWith(['lastPrice', 'highPrice']);
  });

  it('reset button calls onReset', async () => {
    const user = userEvent.setup();
    const onReset = vi.fn();
    renderWithTheme(
      <MetricColumnPicker
        available={metricsForSurface('watchlist')}
        value={['lastPrice']}
        onChange={vi.fn()}
        onReset={onReset}
        getLabel={(k) => k}
        buttonLabel="Columns"
        resetLabel="Reset columns"
      />,
    );
    await user.click(screen.getByRole('button', { name: /reset columns/i }));
    expect(onReset).toHaveBeenCalled();
  });
});
