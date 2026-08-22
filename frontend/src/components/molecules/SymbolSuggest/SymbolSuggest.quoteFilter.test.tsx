import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { SymbolSuggest } from './SymbolSuggest';

const fetchSpot = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useLazyListSpotMarketsQuery: () => [
      fetchSpot,
      {
        data: { items: [] },
        currentData: { items: [] },
        originalArgs: { exchange: 'binance' },
        isFetching: false,
      },
    ],
  };
});

describe('SymbolSuggest quote filter', () => {
  it('does not pin search to the venue default quote (ETHBTC must be findable on Binance)', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SymbolSuggest exchange="binance" value="" onChange={() => undefined} aria-label="Symbol" />,
    );

    await user.type(screen.getByRole('combobox', { name: 'Symbol' }), 'ETHBTC');
    await vi.waitFor(() => {
      expect(fetchSpot).toHaveBeenCalled();
    });

    const arg = fetchSpot.mock.calls.at(-1)?.[0] as { q?: string; quote?: string };
    expect(arg.q).toMatch(/ETHBTC/i);
    // Constraining to USDT hides BTC-quoted pairs from paper/alert/jump search.
    expect(arg.quote).toBeUndefined();
  });
});
