import { configureStore } from '@reduxjs/toolkit';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Provider } from 'react-redux';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { baseApi } from '@/libs/api/baseApi';
import { SymbolSuggest } from './SymbolSuggest';

vi.mock('@/config/env', () => ({
  env: { apiBaseUrl: 'http://rtk.test', apiBaseUrlLabel: 'rtk.test', clientId: 'test-client' },
}));

function makeStore() {
  return configureStore({
    reducer: { [baseApi.reducerPath]: baseApi.reducer },
    middleware: (gdm) => gdm().concat(baseApi.middleware),
  });
}

function requestURL(input: RequestInfo | URL): string {
  if (typeof input === 'string') return input;
  if (input instanceof URL) return input.href;
  if (input instanceof Request) return input.url;
  return String(input);
}

function delayedJSON(body: unknown, ms: number) {
  return new Promise<Response>((resolve) => {
    setTimeout(() => {
      resolve(
        new Response(JSON.stringify(body), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      );
    }, ms);
  });
}

describe('SymbolSuggest venue change', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('does not offer the previous venue’s symbols after exchange changes', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = requestURL(input);
      if (url.includes('exchange=binance')) {
        return delayedJSON({ items: [{ symbol: 'BTCUSDT' }] }, 0);
      }
      if (url.includes('exchange=coinbase')) {
        return delayedJSON({ items: [{ symbol: 'BTC-USD' }] }, 80);
      }
      return delayedJSON({ items: [] }, 0);
    });

    const store = makeStore();
    const user = userEvent.setup();
    const { rerender } = render(
      <Provider store={store}>
        <SymbolSuggest exchange="binance" value="BTC" onChange={() => undefined} aria-label="Symbol" />
      </Provider>,
    );

    await user.click(screen.getByRole('combobox', { name: 'Symbol' }));
    await waitFor(() => expect(screen.getByText('BTC/USDT')).toBeInTheDocument());

    rerender(
      <Provider store={store}>
        <SymbolSuggest exchange="coinbase" value="BTC" onChange={() => undefined} aria-label="Symbol" />
      </Provider>,
    );

    await user.click(screen.getByRole('combobox', { name: 'Symbol' }));
    await waitFor(() => expect(screen.queryByText('BTC/USDT')).not.toBeInTheDocument());
    await waitFor(() => expect(screen.getByText('BTC/USD')).toBeInTheDocument());
  });
});
