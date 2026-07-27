import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, act, fireEvent } from '@testing-library/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { WATCHLIST_STORAGE_KEY } from '@/config/watchlistConstants';
import { serializeLocalWatchlist } from '@/libs/utils';
import { createTestStorage } from '@/libs/utils/storage';
import { resetClientIdCacheForTests } from '@/libs/utils/clientId';

const unwrapGet = vi.fn();
const unwrapAdd = vi.fn();
const unwrapRemove = vi.fn();
const unwrapReplace = vi.fn();

vi.mock('@/libs/api', async () => {
  const actual = await vi.importActual<typeof import('@/libs/api')>('@/libs/api');
  return {
    ...actual,
    useLazyGetWatchlistQuery: () => [
      () => ({ unwrap: unwrapGet }),
    ],
    useAddWatchlistItemMutation: () => [
      () => ({ unwrap: unwrapAdd }),
    ],
    useRemoveWatchlistItemMutation: () => [
      () => ({ unwrap: unwrapRemove }),
    ],
    useReplaceWatchlistMutation: () => [
      () => ({ unwrap: unwrapReplace }),
    ],
    rtkErrorMessage: actual.rtkErrorMessage,
  };
});

// Patch appStorage used by context via module mock
const testStorage = createTestStorage();
vi.mock('@/libs/utils', async () => {
  const actual = await vi.importActual<typeof import('@/libs/utils')>('@/libs/utils');
  return {
    ...actual,
    get appStorage() {
      return testStorage;
    },
    getOrCreateClientId: () => 'mobile-test-client',
  };
});

import { WatchlistProvider, useWatchlist } from './WatchlistContext';

function Probe() {
  const wl = useWatchlist();
  return (
    <div>
      <span data-testid="ready">{String(wl.isReady)}</span>
      <span data-testid="count">{wl.count}</span>
      <span data-testid="error">{wl.error ?? ''}</span>
      <span data-testid="action">{wl.actionError ?? ''}</span>
      <span data-testid="watched">
        {String(wl.isWatched('binance', 'BTCUSDT'))}
      </span>
      <button type="button" onClick={() => void wl.toggle('binance', 'BTCUSDT')}>
        toggle
      </button>
    </div>
  );
}

function renderProbe() {
  // Minimal store so Provider from react-redux is happy if needed
  const store = configureStore({ reducer: { _: (s = {}) => s } });
  return render(
    <Provider store={store}>
      <WatchlistProvider>
        <Probe />
      </WatchlistProvider>
    </Provider>,
  );
}

describe('WatchlistProvider', () => {
  beforeEach(() => {
    resetClientIdCacheForTests();
    testStorage.removeItem(WATCHLIST_STORAGE_KEY);
    unwrapGet.mockReset();
    unwrapAdd.mockReset();
    unwrapRemove.mockReset();
    unwrapReplace.mockReset();
    unwrapGet.mockResolvedValue({ clientId: 'mobile-test-client', items: [] });
    unwrapAdd.mockResolvedValue({ clientId: 'mobile-test-client', items: [] });
    unwrapRemove.mockResolvedValue({ clientId: 'mobile-test-client', items: [] });
    unwrapReplace.mockResolvedValue({ clientId: 'mobile-test-client', items: [] });
  });

  it('hydrates empty list from server', async () => {
    renderProbe();
    await waitFor(() => expect(screen.getByTestId('ready').textContent).toBe('true'));
    expect(screen.getByTestId('count').textContent).toBe('0');
    expect(unwrapGet).toHaveBeenCalled();
  });

  it('merges local favorites when server is empty', async () => {
    testStorage.setItem(
      WATCHLIST_STORAGE_KEY,
      serializeLocalWatchlist([{ exchange: 'binance', symbol: 'BTCUSDT' }]),
    );
    unwrapGet.mockResolvedValue({ clientId: 'c', items: [] });
    renderProbe();
    await waitFor(() => expect(screen.getByTestId('count').textContent).toBe('1'));
    expect(screen.getByTestId('watched').textContent).toBe('true');
    // re-sync missing to server
    await waitFor(() => expect(unwrapReplace).toHaveBeenCalled());
  });

  it('keeps local list when GET fails', async () => {
    testStorage.setItem(
      WATCHLIST_STORAGE_KEY,
      serializeLocalWatchlist([{ exchange: 'binance', symbol: 'ETHUSDT' }]),
    );
    unwrapGet.mockRejectedValue({ status: 'FETCH_ERROR' });
    renderProbe();
    await waitFor(() => expect(screen.getByTestId('ready').textContent).toBe('true'));
    expect(screen.getByTestId('count').textContent).toBe('1');
    expect(screen.getByTestId('error').textContent).toMatch(/Network/);
  });

  it('optimistic add and remove', async () => {
    renderProbe();
    await waitFor(() => expect(screen.getByTestId('ready').textContent).toBe('true'));

    await act(async () => {
      fireEvent.click(screen.getByText('toggle'));
    });
    await waitFor(() => expect(screen.getByTestId('count').textContent).toBe('1'));
    expect(unwrapAdd).toHaveBeenCalled();

    await act(async () => {
      fireEvent.click(screen.getByText('toggle'));
    });
    await waitFor(() => expect(screen.getByTestId('count').textContent).toBe('0'));
    expect(unwrapRemove).toHaveBeenCalled();
  });

  it('rolls back add on mutation failure', async () => {
    unwrapAdd.mockRejectedValue({ status: 500 });
    renderProbe();
    await waitFor(() => expect(screen.getByTestId('ready').textContent).toBe('true'));

    await act(async () => {
      fireEvent.click(screen.getByText('toggle'));
    });
    await waitFor(() => expect(screen.getByTestId('count').textContent).toBe('0'));
    expect((screen.getByTestId('action').textContent ?? '').length).toBeGreaterThan(0);
  });
});
