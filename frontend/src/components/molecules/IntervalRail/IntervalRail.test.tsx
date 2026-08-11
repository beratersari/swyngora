import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { IntervalRail } from './IntervalRail';

describe('IntervalRail', () => {
  it('marks the active interval and fires onChange', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderWithTheme(
      <IntervalRail intervals={['15m', '1h', '4h']} value="1h" onChange={onChange} aria-label="Interval" />,
    );
    expect(screen.getByRole('radio', { name: '1h' })).toHaveAttribute('aria-checked', 'true');
    await user.click(screen.getByRole('radio', { name: '4h' }));
    expect(onChange).toHaveBeenCalledWith('4h');
  });
});
