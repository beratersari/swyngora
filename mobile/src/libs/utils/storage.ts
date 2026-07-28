/**
 * Thin storage adapter — localStorage on web; interface ready for AsyncStorage.
 */

export type KeyValueStorage = {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
};

function createMemoryStorage(): KeyValueStorage {
  const map = new Map<string, string>();
  return {
    getItem: (key) => map.get(key) ?? null,
    setItem: (key, value) => {
      map.set(key, value);
    },
    removeItem: (key) => {
      map.delete(key);
    },
  };
}

function createWebStorage(): KeyValueStorage {
  try {
    if (typeof globalThis !== 'undefined' && 'localStorage' in globalThis && globalThis.localStorage) {
      return globalThis.localStorage;
    }
  } catch {
    // private mode / denied
  }
  return createMemoryStorage();
}

/** Default app storage (web localStorage when available). */
export const appStorage: KeyValueStorage = createWebStorage();

/** Test helper: in-memory storage. */
export function createTestStorage(initial?: Record<string, string>): KeyValueStorage {
  const mem = createMemoryStorage();
  if (initial) {
    for (const [k, v] of Object.entries(initial)) {
      mem.setItem(k, v);
    }
  }
  return mem;
}
