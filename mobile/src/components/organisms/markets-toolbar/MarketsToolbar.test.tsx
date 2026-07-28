import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MarketsToolbar } from './MarketsToolbar';

describe('MarketsToolbar', () => {
  it('renders filters and favorites toggle', () => {
    const onToggle = vi.fn();
    render(
      <MarketsToolbar
        search=""
        onSearchChange={vi.fn()}
        activeFilterCount={0}
        onOpenFilters={vi.fn()}
        favoritesOnly={false}
        onToggleFavoritesOnly={onToggle}
        favoritesCount={2}
      />,
    );
    expect(screen.getByText('Filters')).toBeTruthy();
    expect(screen.getByText(/Favorites \(2\)/)).toBeTruthy();
    fireEvent.click(screen.getByLabelText('Show favorites only'));
    expect(onToggle).toHaveBeenCalled();
  });

  it('shows favorites-only active label', () => {
    render(
      <MarketsToolbar
        search=""
        onSearchChange={vi.fn()}
        activeFilterCount={1}
        onOpenFilters={vi.fn()}
        favoritesOnly
        onToggleFavoritesOnly={vi.fn()}
        favoritesCount={1}
      />,
    );
    expect(screen.getByText(/Favorites only \(1\)/)).toBeTruthy();
    expect(screen.getByText('Filters (1)')).toBeTruthy();
  });
});
