import { useRealtimeConnection } from '@/libs/realtime';

/** Mount once under Redux so live ticks patch RTK Query caches. */
export function RealtimeBridge() {
  useRealtimeConnection();
  return null;
}
