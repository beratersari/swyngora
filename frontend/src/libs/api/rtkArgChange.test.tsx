import { configureStore } from '@reduxjs/toolkit';
import { renderHook, waitFor } from '@testing-library/react';
import { Provider } from 'react-redux';
import type { ReactNode } from 'react';

vi.mock('@/config/env', () => ({
  env: { apiBaseUrl: 'http://rtk.test', apiBaseUrlLabel: 'rtk.test', clientId: 'test-client' },
}));

import { baseApi } from './baseApi';
import { useGetTicker24hQuery, useGetPortfolioQuery } from './index';
import { rtkCurrent, rtkCurrentPending } from '@/libs/utils';

function makeStore() {
  return configureStore({
    reducer: { [baseApi.reducerPath]: baseApi.reducer },
    middleware: (gdm) => gdm().concat(baseApi.middleware),
  });
}

function wrap(store: ReturnType<typeof makeStore>) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <Provider store={store}>{children}</Provider>;
  };
}

function requestURL(input: RequestInfo | URL): string {
  if (typeof input === 'string') return input;
  if (input instanceof URL) return input.href;
  if (typeof Request !== 'undefined' && input instanceof Request) return input.url;
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

describe('RTK Query current-arg data', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('does not treat previous book as live while the next book loads', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = requestURL(input);
      if (url.includes('portfolioId=book-a')) {
        return delayedJSON({ id: 'book-a', cashBalance: 10000, availableCash: 10000 }, 0);
      }
      if (url.includes('portfolioId=book-b')) {
        return delayedJSON({ id: 'book-b', cashBalance: 1, availableCash: 1 }, 80);
      }
      return delayedJSON({}, 0);
    });

    const store = makeStore();
    const { result, rerender } = renderHook(
      ({ id }: { id: string }) => useGetPortfolioQuery({ portfolioId: id }),
      { wrapper: wrap(store), initialProps: { id: 'book-a' } },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(rtkCurrent(result.current)?.id).toBe('book-a');

    rerender({ id: 'book-b' });
    await waitFor(() => expect(result.current.originalArgs).toEqual({ portfolioId: 'book-b' }));
    expect(rtkCurrent(result.current)).toBeUndefined();
    expect(rtkCurrentPending(result.current)).toBe(true);
    expect(result.current.data?.id).toBe('book-a');
  });

  it('does not treat previous ticker as live after symbol change', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = requestURL(input);
      if (url.includes('symbol=BTCUSDT')) {
        return delayedJSON({ symbol: 'BTCUSDT', lastPrice: '67000.00' }, 0);
      }
      if (url.includes('symbol=ETHUSDT')) {
        return delayedJSON({ symbol: 'ETHUSDT', lastPrice: '3200.00' }, 80);
      }
      return delayedJSON({}, 0);
    });

    const store = makeStore();
    const { result, rerender } = renderHook(
      ({ symbol }: { symbol: string }) => useGetTicker24hQuery({ exchange: 'binance', symbol }),
      { wrapper: wrap(store), initialProps: { symbol: 'BTCUSDT' } },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    rerender({ symbol: 'ETHUSDT' });
    await waitFor(() => expect(result.current.originalArgs?.symbol).toBe('ETHUSDT'));
    expect(rtkCurrent(result.current)).toBeUndefined();
    expect(result.current.data?.lastPrice).toBe('67000.00');
  });
});
