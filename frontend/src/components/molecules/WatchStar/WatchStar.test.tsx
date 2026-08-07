import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { WatchStar } from './WatchStar';

describe('WatchStar', () => {
  it('toggles aria label and fires click', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    const { rerender } = renderWithTheme(
      <WatchStar watched={false} addLabel="Add" removeLabel="Remove" onClick={onClick} />,
    );
    expect(screen.getByRole('button', { name: 'Add' })).toHaveAttribute('aria-pressed', 'false');
    await user.click(screen.getByRole('button', { name: 'Add' }));
    expect(onClick).toHaveBeenCalled();
    rerender(<WatchStar watched addLabel="Add" removeLabel="Remove" onClick={onClick} />);
    expect(screen.getByRole('button', { name: 'Remove' })).toHaveAttribute('aria-pressed', 'true');
  });
});
