import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SearchField } from './SearchField';

describe('SearchField', () => {
  it('calls onChangeText', () => {
    const onChange = vi.fn();
    render(<SearchField value="" onChangeText={onChange} accessibilityLabel="Search markets" />);
    const input = screen.getByLabelText('Search markets');
    fireEvent.change(input, { target: { value: 'btc' } });
    expect(onChange).toHaveBeenCalled();
  });
});
