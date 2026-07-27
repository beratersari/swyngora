import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { IntervalToolbar } from './IntervalToolbar';

describe('IntervalToolbar', () => {
  it('renders intervals and selects', () => {
    const onSelect = vi.fn();
    render(
      <IntervalToolbar
        intervals={['1h', '4h']}
        selected="1h"
        onSelect={onSelect}
        showEma
        onToggleEma={vi.fn()}
      />,
    );
    expect(screen.getByText('Interval')).toBeTruthy();
    fireEvent.click(screen.getByText('4h'));
    expect(onSelect).toHaveBeenCalledWith('4h');
  });

  it('toggles EMA chip', () => {
    const onToggle = vi.fn();
    render(
      <IntervalToolbar
        intervals={['1h']}
        selected="1h"
        onSelect={vi.fn()}
        showEma={false}
        onToggleEma={onToggle}
      />,
    );
    fireEvent.click(screen.getByText('EMA off'));
    expect(onToggle).toHaveBeenCalled();
  });
});
