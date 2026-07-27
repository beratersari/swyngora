import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Star } from 'lucide-react-native';
import { Icon } from './Icon';

describe('Icon', () => {
  it('renders lucide icon with accessibility label', () => {
    render(<Icon icon={Star} accessibilityLabel="Favorite" />);
    expect(screen.getByLabelText('Favorite')).toBeTruthy();
  });
});
