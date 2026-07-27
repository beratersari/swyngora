import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ChipGroup } from './ChipGroup';

describe('ChipGroup', () => {
  it('shows empty label when no options', () => {
    render(
      <ChipGroup
        options={[]}
        selected=""
        onSelect={vi.fn()}
        emptyLabel="No tags"
      />,
    );
    expect(screen.getByText('No tags')).toBeTruthy();
  });

  it('selects single option', () => {
    const onSelect = vi.fn();
    render(
      <ChipGroup
        options={[
          { value: 'a', label: 'Alpha' },
          { value: 'b', label: 'Beta' },
        ]}
        selected="a"
        onSelect={onSelect}
        mode="single"
      />,
    );
    fireEvent.click(screen.getByText('Beta'));
    expect(onSelect).toHaveBeenCalledWith('b');
  });

  it('supports multi selected state', () => {
    render(
      <ChipGroup
        options={[
          { value: 'Meme', label: 'Meme' },
          { value: 'AI', label: 'AI' },
        ]}
        selected={['Meme']}
        onSelect={vi.fn()}
        mode="multi"
      />,
    );
    expect(screen.getByText('Meme')).toBeTruthy();
    expect(screen.getByText('AI')).toBeTruthy();
  });
});
