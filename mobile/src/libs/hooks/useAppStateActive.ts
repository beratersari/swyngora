import { useEffect, useState } from 'react';
import { AppState, type AppStateStatus } from 'react-native';

/**
 * True when the app is in the foreground (native) or the document is visible (web).
 * Used to pause RTK Query polling when backgrounded (§6.6).
 */
export function useAppStateActive(): boolean {
  const [active, setActive] = useState(() => AppState.currentState === 'active');

  useEffect(() => {
    const onChange = (next: AppStateStatus) => {
      setActive(next === 'active');
    };
    const sub = AppState.addEventListener('change', onChange);
    return () => sub.remove();
  }, []);

  return active;
}
