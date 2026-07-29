import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CategoriesPage } from './CategoriesPage';
import type { CategoriesPageViewModel } from './CategoriesPage.types';

const stubVm = (): CategoriesPageViewModel => ({
  title: 'Categories',
  search: '',
  onSearchChange: vi.fn(),
  searchPlaceholder: 'Search tags…',
  featuredTitle: 'Featured',
  featuredTags: ['Meme'],
  allTitle: 'All tags',
  tags: ['Meme', 'AI'],
  selectedTag: null,
  isLoading: false,
  isSearchDebouncing: false,
  errorMessage: null,
  emptyMessage: null,
  formatLabel: (t) => t,
  onSelectTag: vi.fn(),
  onRetry: vi.fn(),
  onBack: vi.fn(),
  retryLabel: 'Retry',
  backLabel: 'Back',
});

describe('CategoriesPage', () => {
  it('renders tags from view model and selects', () => {
    const vm = stubVm();
    render(<CategoriesPage viewModel={vm} />);
    expect(screen.getByText('Categories')).toBeTruthy();
    expect(screen.getByText('Featured')).toBeTruthy();
    fireEvent.click(screen.getAllByText('AI')[0]);
    expect(vm.onSelectTag).toHaveBeenCalledWith('AI');
  });

  it('shows error and retry', () => {
    const vm = { ...stubVm(), errorMessage: 'Tags failed', tags: [], featuredTags: [] };
    render(<CategoriesPage viewModel={vm} />);
    expect(screen.getByText('Tags failed')).toBeTruthy();
    fireEvent.click(screen.getByText('Retry'));
    expect(vm.onRetry).toHaveBeenCalled();
  });
});
