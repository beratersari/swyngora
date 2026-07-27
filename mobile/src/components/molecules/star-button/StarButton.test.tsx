import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { StarButton } from './StarButton';

describe('StarButton', () => {
  it('renders add label when not watched', () => {
    render(<StarButton watched={false} onPress={vi.fn()} />);
    expect(screen.getByLabelText('Add to favorites')).toBeTruthy();
  });

  it('renders remove label when watched', () => {
    render(<StarButton watched onPress={vi.fn()} />);
    expect(screen.getByLabelText('Remove from favorites')).toBeTruthy();
  });

  it('calls onPress', () => {
    const onPress = vi.fn();
    render(<StarButton watched={false} onPress={onPress} />);
    fireEvent.click(screen.getByLabelText('Add to favorites'));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it('uses custom accessibility label', () => {
    render(
      <StarButton watched={false} onPress={vi.fn()} accessibilityLabel="Star BTC" />,
    );
    expect(screen.getByLabelText('Star BTC')).toBeTruthy();
  });
});
