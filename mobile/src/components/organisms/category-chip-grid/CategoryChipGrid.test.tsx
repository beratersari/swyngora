import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CategoryChipGrid } from './CategoryChipGrid';

describe('CategoryChipGrid', () => {
  it('renders featured and all tags and selects', () => {
    const onSelect = vi.fn();
    render(
      <CategoryChipGrid
        featuredTitle="Featured"
        featuredTags={['Meme']}
        allTitle="All"
        tags={['Meme', 'AI']}
        onSelectTag={onSelect}
      />,
    );
    expect(screen.getByText('Featured')).toBeTruthy();
    expect(screen.getByText('All')).toBeTruthy();
    fireEvent.click(screen.getAllByText('AI')[0]);
    expect(onSelect).toHaveBeenCalledWith('AI');
  });

  it('shows empty message when no tags and not loading', () => {
    render(
      <CategoryChipGrid
        featuredTags={[]}
        tags={[]}
        emptyMessage="No tags"
        onSelectTag={vi.fn()}
      />,
    );
    expect(screen.getByText('No tags')).toBeTruthy();
  });
});
