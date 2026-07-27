import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MarketsFilterPage } from './MarketsFilterPage';
import type { MarketsFilterPageViewModel } from './MarketsFilterPage.types';

function makeVm(
  overrides: Partial<MarketsFilterPageViewModel> = {},
): MarketsFilterPageViewModel {
  return {
    title: 'Filters',
    quote: 'USDT',
    quoteOptions: ['USDT', 'USD'],
    onQuoteChange: vi.fn(),
    sort: 'quoteVolume',
    order: 'desc',
    sortOptions: [{ value: 'quoteVolume', label: 'Quote vol' }],
    onSortChange: vi.fn(),
    onOrderChange: vi.fn(),
    availableTags: ['defi', 'Meme'],
    draftTags: ['defi'],
    isLoadingTags: false,
    tagsError: null,
    searchTag: '',
    onSearchTagChange: vi.fn(),
    onToggleTag: vi.fn(),
    onClearTags: vi.fn(),
    onSelectAllVisible: vi.fn(),
    onResetAll: vi.fn(),
    onApply: vi.fn(),
    onCancel: vi.fn(),
    selectedTagsCount: 1,
    ...overrides,
  };
}

describe('MarketsFilterPage', () => {
  it('renders quote, sort, tags and apply', () => {
    const onApply = vi.fn();
    render(<MarketsFilterPage viewModel={makeVm({ onApply })} />);
    expect(screen.getByText('Filters')).toBeTruthy();
    expect(screen.getByText('Quote')).toBeTruthy();
    expect(screen.getByText('Sort')).toBeTruthy();
    expect(screen.getByText('USDT')).toBeTruthy();
    expect(screen.getByText('defi')).toBeTruthy();
    fireEvent.click(screen.getByText('Apply'));
    expect(onApply).toHaveBeenCalled();
  });
});
