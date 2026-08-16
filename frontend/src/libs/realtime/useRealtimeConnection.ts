import { useEffect, useState } from 'react';
import { useAppDispatch } from '@/libs/api/hooks';
import { store } from '@/libs/api/store';
import { applyPortfolioEvent, applyPriceTick } from './applyUpdates';
import { getRealtimeClient } from './client';
import { REALTIME_TYPE } from './constants';
import type { RealtimePortfolioEvent, RealtimePriceTick } from './realtime.types';

/** Opens the shared socket and patches RTK Query caches from live events. */
export function useRealtimeConnection(): { connected: boolean } {
  const dispatch = useAppDispatch();
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const client = getRealtimeClient();
    client.start();
    const offStatus = client.onStatus(setConnected);
    const offMsg = client.onMessage((msg) => {
      if (msg.type === REALTIME_TYPE.price) {
        applyPriceTick(dispatch, store.getState, msg as RealtimePriceTick);
      } else if (msg.type === REALTIME_TYPE.portfolio) {
        applyPortfolioEvent(dispatch, msg as RealtimePortfolioEvent, store.getState);
      }
    });
    return () => {
      offStatus();
      offMsg();
    };
  }, [dispatch]);

  return { connected };
}

export function useRealtimeConnected(): boolean {
  const [connected, setConnected] = useState(() => getRealtimeClient().connected);
  useEffect(() => getRealtimeClient().onStatus(setConnected), []);
  return connected;
}
