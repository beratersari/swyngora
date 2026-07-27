import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from './Button';

describe('Button', () => {
  it('renders label and handles press', () => {
    const onPress = vi.fn();
    render(<Button label="Retry" onPress={onPress} />);
    fireEvent.click(screen.getByText('Retry'));
    expect(onPress).toHaveBeenCalled();
  });
});
