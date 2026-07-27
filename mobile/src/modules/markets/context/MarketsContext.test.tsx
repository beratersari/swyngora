import { describe, expect, it } from 'vitest';
import { render, screen, act, fireEvent } from '@testing-library/react';
import { MarketsProvider, useMarketsContext } from './MarketsContext';

function Probe() {
  const m = useMarketsContext();
  return (
    <div>
      <span data-testid="exchange">{m.exchange}</span>
      <span data-testid="search">{m.search}</span>
      <span data-testid="quote">{m.quote}</span>
      <span data-testid="tags">{m.selectedTags.join(',')}</span>
      <span data-testid="filters">{m.activeFilterCount}</span>
      <span data-testid="rev">{m.filterRevision}</span>
      <button type="button" onClick={() => m.setExchange('coinbase')}>
        setEx
      </button>
      <button type="button" onClick={() => m.setSearch('btc')}>
        setSearch
      </button>
      <button
        type="button"
        onClick={() =>
          m.applyListFilters({
            quote: 'USD',
            selectedTags: ['Meme'],
            sort: 'lastPrice',
            order: 'asc',
          })
        }
      >
        apply
      </button>
    </div>
  );
}

describe('MarketsProvider', () => {
  it('defaults and mutates filter state', () => {
    render(
      <MarketsProvider>
        <Probe />
      </MarketsProvider>,
    );
    expect(screen.getByTestId('exchange').textContent).toBe('binance');
    expect(screen.getByTestId('quote').textContent).toBe('USDT');

    act(() => {
      fireEvent.click(screen.getByText('setEx'));
    });
    expect(screen.getByTestId('exchange').textContent).toBe('coinbase');
    expect(screen.getByTestId('tags').textContent).toBe('');

    act(() => {
      fireEvent.click(screen.getByText('setSearch'));
    });
    expect(screen.getByTestId('search').textContent).toBe('btc');

    act(() => {
      fireEvent.click(screen.getByText('apply'));
    });
    expect(screen.getByTestId('quote').textContent).toBe('USD');
    expect(screen.getByTestId('tags').textContent).toBe('Meme');
    expect(Number(screen.getByTestId('filters').textContent)).toBeGreaterThan(0);
    expect(Number(screen.getByTestId('rev').textContent)).toBeGreaterThan(0);
  });
});
