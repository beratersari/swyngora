import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MarketsFilterForm } from './MarketsFilterForm';

const base = {
  quote: 'USDT',
  quoteOptions: ['USDT', 'USD'],
  onQuoteChange: vi.fn(),
  sort: 'quoteVolume',
  order: 'desc' as const,
  sortOptions: [
    { value: 'quoteVolume', label: 'Quote vol' },
    { value: 'lastPrice', label: 'Price' },
  ],
  onSortChange: vi.fn(),
  onOrderChange: vi.fn(),
  availableTags: ['Meme', 'AI'],
  selectedTags: [] as string[],
  isLoadingTags: false,
  tagsError: null as string | null,
  tagSearch: '',
  onTagSearchChange: vi.fn(),
  onToggleTag: vi.fn(),
  onClearTags: vi.fn(),
  onSelectAllVisible: vi.fn(),
  onResetAll: vi.fn(),
};

describe('MarketsFilterForm', () => {
  it('changes quote and sort', () => {
    const onQuoteChange = vi.fn();
    const onSortChange = vi.fn();
    render(
      <MarketsFilterForm
        {...base}
        onQuoteChange={onQuoteChange}
        onSortChange={onSortChange}
      />,
    );
    fireEvent.click(screen.getByText('USD'));
    expect(onQuoteChange).toHaveBeenCalledWith('USD');
    fireEvent.click(screen.getByText('Price'));
    expect(onSortChange).toHaveBeenCalledWith('lastPrice');
  });

  it('fires tag actions', () => {
    const onClearTags = vi.fn();
    const onResetAll = vi.fn();
    const onToggleTag = vi.fn();
    render(
      <MarketsFilterForm
        {...base}
        onClearTags={onClearTags}
        onResetAll={onResetAll}
        onToggleTag={onToggleTag}
      />,
    );
    fireEvent.click(screen.getByText('Clear tags'));
    expect(onClearTags).toHaveBeenCalled();
    fireEvent.click(screen.getByText('Reset all filters'));
    expect(onResetAll).toHaveBeenCalled();
    fireEvent.click(screen.getByText('Meme'));
    expect(onToggleTag).toHaveBeenCalledWith('Meme');
  });

  it('shows tags error', () => {
    render(<MarketsFilterForm {...base} tagsError="tags failed" />);
    expect(screen.getByText('tags failed')).toBeTruthy();
  });
});
