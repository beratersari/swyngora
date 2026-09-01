import { useEffect, useRef } from 'react';

/** Revalidate when the app returns to the foreground (AGENTS.md §6.6). */
export function useRefetchOnResume(refetch: () => unknown, active: boolean): void {
  const prev = useRef(active);
  useEffect(() => {
    if (active && !prev.current) {
      void refetch();
    }
    prev.current = active;
  }, [active, refetch]);
}
