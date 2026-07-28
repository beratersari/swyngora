import {
  CLIENT_ID_PREFIX,
  CLIENT_ID_STORAGE_KEY,
} from '@/config/watchlistConstants';
import { appStorage, type KeyValueStorage } from './storage';

let memoryCache: string | null = null;

function isForbiddenClientId(id: string): boolean {
  const t = id.trim();
  return t === '' || t.toLowerCase() === 'default';
}

function randomUuid(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  // Fallback for older environments / some test runners
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

/**
 * Returns a stable non-empty client id (never "default").
 * Caches in memory + storage.
 */
export function getOrCreateClientId(storage: KeyValueStorage = appStorage): string {
  if (memoryCache && !isForbiddenClientId(memoryCache)) {
    return memoryCache;
  }
  const existing = storage.getItem(CLIENT_ID_STORAGE_KEY)?.trim() ?? '';
  if (existing && !isForbiddenClientId(existing)) {
    memoryCache = existing;
    return existing;
  }
  const next = `${CLIENT_ID_PREFIX}${randomUuid()}`;
  storage.setItem(CLIENT_ID_STORAGE_KEY, next);
  memoryCache = next;
  return next;
}

/** Cached id if already created in this session / storage; null if never set. */
export function peekClientId(storage: KeyValueStorage = appStorage): string | null {
  if (memoryCache && !isForbiddenClientId(memoryCache)) {
    return memoryCache;
  }
  const existing = storage.getItem(CLIENT_ID_STORAGE_KEY)?.trim() ?? '';
  if (existing && !isForbiddenClientId(existing)) {
    memoryCache = existing;
    return existing;
  }
  return null;
}

/** Test-only: clear in-memory cache. */
export function resetClientIdCacheForTests(): void {
  memoryCache = null;
}
