/** RTK Query result fields needed to read the current-arg payload. */
export type RtkQuerySlice<T> = {
  currentData?: T;
  data?: T;
  isFetching?: boolean;
  isLoading?: boolean;
};

/**
 * Payload for the hook’s current args.
 * RTK `.data` stays on the last successful result for any args while a new
 * request is in flight. Pages must not treat that as the live symbol/venue.
 */
export function rtkCurrent<T>(q: RtkQuerySlice<T> | undefined): T | undefined {
  return q?.currentData;
}

/** True while the current args have no result yet (including arg change). */
export function rtkCurrentPending(q: RtkQuerySlice<unknown> | undefined): boolean {
  if (!q) return false;
  return Boolean(q.isFetching || q.isLoading) && q.currentData === undefined;
}
