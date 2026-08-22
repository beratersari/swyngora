import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { SymbolSuggest } from './SymbolSuggest';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useLazyListSpotMarketsQuery: () => [
      vi.fn(),
      {
        data: { items: [{ symbol: 'BTCUSDT', baseAsset: 'BTC' }] },
        currentData: { items: [{ symbol: 'BTCUSDT', baseAsset: 'BTC' }] },
        originalArgs: { exchange: 'binance' },
        isFetching: false,
      },
    ],
  };
});

describe('SymbolSuggest', () => {
  it('renders combobox with value', () => {
    renderWithProviders(
      <SymbolSuggest exchange="binance" value="BTC" onChange={() => undefined} aria-label="Symbol" />,
    );
    expect(screen.getByRole('combobox', { name: 'Symbol' })).toBeInTheDocument();
  });
});
