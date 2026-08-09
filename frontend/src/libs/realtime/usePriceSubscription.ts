import { useEffect, useMemo } from 'react';
import { getRealtimeClient } from './client';
import { symbolKey, uniqueSymbolRefs } from './helpers';
import { useRealtimeConnected } from './useRealtimeConnection';
import type { RealtimeSymbolRef } from './realtime.types';

/** Subscribe to live prices for the given coins. Refcounted; resubscribes after reconnect. */
export function usePriceSubscription(symbols: RealtimeSymbolRef[], enabled = true): { connected: boolean } {
  const connected = useRealtimeConnected();
  const key = symbolsKey(symbols);
  const normalized = useMemo(() => uniqueSymbolRefs(symbols), [key]);

  useEffect(() => {
    if (!enabled || normalized.length === 0) return;
    const client = getRealtimeClient();
    client.subscribePrices(normalized);
    return () => client.unsubscribePrices(normalized);
  }, [enabled, normalized]);

  return { connected };
}

function symbolsKey(symbols: RealtimeSymbolRef[]): string {
  return uniqueSymbolRefs(symbols)
    .map((s) => symbolKey(s))
    .sort()
    .join(',');
}
