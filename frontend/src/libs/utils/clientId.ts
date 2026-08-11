import { env } from '@/config/env';

const STORAGE_KEY = 'swyngora.clientId';

function randomId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `c-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

/**
 * Stable client tenancy key for watchlist APIs.
 * Prefer VITE_CLIENT_ID; otherwise persist a generated id in localStorage.
 */
export function getOrCreateClientId(): string {
  const fromEnv = env.clientId?.trim();
  if (fromEnv) return fromEnv;
  if (typeof localStorage === 'undefined') {
    throw new Error('clientId storage unavailable');
  }
  try {
    const existing = localStorage.getItem(STORAGE_KEY)?.trim();
    if (existing) return existing;
    const created = randomId();
    localStorage.setItem(STORAGE_KEY, created);
    return created;
  } catch {
    throw new Error('clientId storage unavailable');
  }
}
