import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Chip } from './Chip';

describe('Chip', () => {
  it('renders and fires press', () => {
    const onPress = vi.fn();
    render(<Chip label="USDT" onPress={onPress} active />);
    fireEvent.click(screen.getByText('USDT'));
    expect(onPress).toHaveBeenCalled();
  });
});
