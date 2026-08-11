import { getOrCreateClientId } from './clientId';

const STORAGE_KEY = 'swyngora.aiSessionId';

function randomSuffix(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID().slice(0, 8);
  }
  return Math.random().toString(36).slice(2, 10);
}

/**
 * Stable multi-turn session id for POST /api/v1/ai/chat and /chat/stream.
 * Prefixed so server logs can distinguish web from other clients.
 */
export function getOrCreateAiSessionId(): string {
  if (typeof localStorage === 'undefined') {
    return `web-ai-${getOrCreateClientId()}`;
  }
  try {
    const existing = localStorage.getItem(STORAGE_KEY)?.trim();
    if (existing) return existing;
    const created = `web-ai-${getOrCreateClientId()}-${randomSuffix()}`;
    localStorage.setItem(STORAGE_KEY, created);
    return created;
  } catch {
    return `web-ai-${getOrCreateClientId()}`;
  }
}

/** Start a fresh conversation thread (new session id). */
export function resetAiSessionId(): string {
  const created = `web-ai-${getOrCreateClientId()}-${randomSuffix()}`;
  persistAiSessionId(created);
  return created;
}

/** Persist a server- or client-issued session id for multi-turn continuity. */
export function persistAiSessionId(id: string): void {
  const trimmed = id.trim();
  if (!trimmed) return;
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, trimmed);
    }
  } catch {
    /* ignore quota / private mode */
  }
}
