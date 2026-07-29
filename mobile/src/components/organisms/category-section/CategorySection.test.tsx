import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CategorySection } from './CategorySection';

describe('CategorySection', () => {
  it('renders tags and invokes select', () => {
    const onSelect = vi.fn();
    render(
      <CategorySection
        title="Categories"
        tags={['Meme', 'AI']}
        onSelectTag={onSelect}
        actionLabel="See all"
        onAction={vi.fn()}
      />,
    );
    expect(screen.getByText('Categories')).toBeTruthy();
    expect(screen.getByText('See all')).toBeTruthy();
    fireEvent.click(screen.getByText('Meme'));
    expect(onSelect).toHaveBeenCalledWith('Meme');
  });

  it('hides when empty and no empty message', () => {
    const { container } = render(
      <CategorySection title="Categories" tags={[]} onSelectTag={vi.fn()} />,
    );
    expect(container.textContent).toBe('');
  });
});
