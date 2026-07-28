import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { MAX_WATCHLIST_ITEMS, WATCHLIST_STORAGE_KEY } from '@/config/watchlistConstants';
import {
  rtkErrorMessage,
  useAddWatchlistItemMutation,
  useLazyGetWatchlistQuery,
  useRemoveWatchlistItemMutation,
  useReplaceWatchlistMutation,
} from '@/libs/api';
import { i18n } from '@/libs/i18n';
import {
  appStorage,
  getOrCreateClientId,
  isAtMaxItems,
  mergeWatchlists,
  normalizePair,
  readLocalWatchlist,
  serializeLocalWatchlist,
  watchKey,
  type WatchlistPair,
} from '@/libs/utils';
import type { WatchlistContextValue } from './WatchlistContext.types';

const WatchlistContext = createContext<WatchlistContextValue | null>(null);

function persistLocal(items: WatchlistPair[]) {
  appStorage.setItem(WATCHLIST_STORAGE_KEY, serializeLocalWatchlist(items));
}

export function WatchlistProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<WatchlistPair[]>([]);
  const [isReady, setIsReady] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const hydratedRef = useRef(false);

  const [fetchWatchlist] = useLazyGetWatchlistQuery();
  const [addItem] = useAddWatchlistItemMutation();
  const [removeItem] = useRemoveWatchlistItemMutation();
  const [replaceList] = useReplaceWatchlistMutation();

  const membership = useMemo(() => {
    const set = new Set<string>();
    for (const i of items) {
      set.add(watchKey(i.exchange, i.symbol));
    }
    return set;
  }, [items]);

  const applyItems = useCallback((next: WatchlistPair[]) => {
    setItems(next);
    persistLocal(next);
  }, []);

  const hydrate = useCallback(async () => {
    // Ensure client id exists before API calls
    getOrCreateClientId();
    const local = readLocalWatchlist(appStorage.getItem(WATCHLIST_STORAGE_KEY));

    try {
      const server = await fetchWatchlist().unwrap();
      const serverPairs: WatchlistPair[] = (server.items ?? []).map((i) =>
        normalizePair(i.exchange ?? 'binance', i.symbol ?? ''),
      ).filter((p) => p.symbol);

      const merged = mergeWatchlists(local, serverPairs);
      applyItems(merged);
      setError(null);

      // Re-sync items server is missing (e.g. after backend restart)
      const serverKeys = new Set(serverPairs.map((p) => watchKey(p.exchange, p.symbol)));
      const missing = merged.filter((p) => !serverKeys.has(watchKey(p.exchange, p.symbol)));
      if (missing.length > 0) {
        try {
          await replaceList({
            items: merged.map((m) => ({
              exchange: m.exchange,
              symbol: m.symbol,
              ...(m.note ? { note: m.note } : {}),
            })),
          }).unwrap();
        } catch {
          // Keep local; server may be rate-limited — next refresh will retry
        }
      }
    } catch (e) {
      // Offline / server down: keep local
      applyItems(local);
      setError(rtkErrorMessage(e, { resource: 'watchlist' }));
    } finally {
      setIsReady(true);
    }
  }, [applyItems, fetchWatchlist, replaceList]);

  useEffect(() => {
    if (hydratedRef.current) return;
    hydratedRef.current = true;
    void hydrate();
  }, [hydrate]);

  const isWatched = useCallback(
    (exchange: string, symbol: string) => membership.has(watchKey(exchange, symbol)),
    [membership],
  );

  const toggle = useCallback(
    async (exchange: string, symbol: string) => {
      const pair = normalizePair(exchange, symbol);
      if (!pair.symbol) return;

      const key = watchKey(pair.exchange, pair.symbol);
      const currently = membership.has(key);
      setActionError(null);

      if (!currently && isAtMaxItems(items, MAX_WATCHLIST_ITEMS)) {
        setActionError(i18n.t('watchlist:full', { max: MAX_WATCHLIST_ITEMS }));
        return;
      }

      const prev = items;
      if (currently) {
        applyItems(items.filter((i) => watchKey(i.exchange, i.symbol) !== key));
        try {
          await removeItem({ exchange: pair.exchange, symbol: pair.symbol }).unwrap();
        } catch (e) {
          applyItems(prev);
          setActionError(rtkErrorMessage(e, { resource: 'watchlist' }));
        }
      } else {
        applyItems([...items, pair]);
        try {
          await addItem({ exchange: pair.exchange, symbol: pair.symbol }).unwrap();
        } catch (e) {
          applyItems(prev);
          setActionError(rtkErrorMessage(e, { resource: 'watchlist' }));
        }
      }
    },
    [addItem, applyItems, items, membership, removeItem],
  );

  const refresh = useCallback(async () => {
    await hydrate();
  }, [hydrate]);

  const clearActionError = useCallback(() => setActionError(null), []);

  const value = useMemo<WatchlistContextValue>(
    () => ({
      isReady,
      items,
      count: items.length,
      error,
      actionError,
      clearActionError,
      isWatched,
      toggle,
      refresh,
    }),
    [
      isReady,
      items,
      error,
      actionError,
      clearActionError,
      isWatched,
      toggle,
      refresh,
    ],
  );

  return (
    <WatchlistContext.Provider value={value}>{children}</WatchlistContext.Provider>
  );
}

export function useWatchlist(): WatchlistContextValue {
  const ctx = useContext(WatchlistContext);
  if (!ctx) {
    throw new Error('useWatchlist must be used within WatchlistProvider');
  }
  return ctx;
}

export function useOptionalWatchlist(): WatchlistContextValue | null {
  return useContext(WatchlistContext);
}
