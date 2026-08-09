import { useEffect } from 'react';
import { getRealtimeClient } from './client';
import { useRealtimeConnected } from './useRealtimeConnection';

/** Subscribe to order/position/cash events for one paper book the user can access. */
export function usePortfolioSubscription(portfolioId: string | undefined, enabled = true): {
  connected: boolean;
} {
  const connected = useRealtimeConnected();
  const id = (portfolioId ?? '').trim();

  useEffect(() => {
    if (!enabled || !id) return;
    const client = getRealtimeClient();
    client.subscribePortfolio(id);
    return () => client.unsubscribePortfolio(id);
  }, [enabled, id]);

  return { connected };
}
