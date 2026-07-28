import {
  AI_SESSION_ID_PREFIX,
  AI_SESSION_STORAGE_KEY,
} from '@/config/aiChatConstants';
import { appStorage, type KeyValueStorage } from './storage';

let memoryCache: string | null = null;

function isForbiddenSessionId(id: string): boolean {
  const t = id.trim().toLowerCase();
  return t === '' || t === 'default' || t === 'http-default';
}

function randomUuid(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

/** Stable multi-turn session id for this device (never empty / default / http-default). */
export function getOrCreateAiSessionId(storage: KeyValueStorage = appStorage): string {
  if (memoryCache && !isForbiddenSessionId(memoryCache)) {
    return memoryCache;
  }
  const existing = storage.getItem(AI_SESSION_STORAGE_KEY)?.trim() ?? '';
  if (existing && !isForbiddenSessionId(existing)) {
    memoryCache = existing;
    return existing;
  }
  const next = `${AI_SESSION_ID_PREFIX}${randomUuid()}`;
  storage.setItem(AI_SESSION_STORAGE_KEY, next);
  memoryCache = next;
  return next;
}

/** Rotate session (new chat). */
export function rotateAiSessionId(storage: KeyValueStorage = appStorage): string {
  const next = `${AI_SESSION_ID_PREFIX}${randomUuid()}`;
  storage.setItem(AI_SESSION_STORAGE_KEY, next);
  memoryCache = next;
  return next;
}

export function peekAiSessionId(storage: KeyValueStorage = appStorage): string | null {
  if (memoryCache && !isForbiddenSessionId(memoryCache)) {
    return memoryCache;
  }
  const existing = storage.getItem(AI_SESSION_STORAGE_KEY)?.trim() ?? '';
  if (existing && !isForbiddenSessionId(existing)) {
    memoryCache = existing;
    return existing;
  }
  return null;
}

/** Test-only. */
export function resetAiSessionCacheForTests(): void {
  memoryCache = null;
}
