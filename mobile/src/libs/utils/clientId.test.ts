import { describe, expect, it, beforeEach } from 'vitest';
import { CLIENT_ID_PREFIX, CLIENT_ID_STORAGE_KEY } from '@/config/watchlistConstants';
import {
  getOrCreateClientId,
  peekClientId,
  resetClientIdCacheForTests,
} from './clientId';
import { createTestStorage } from './storage';

describe('clientId', () => {
  beforeEach(() => {
    resetClientIdCacheForTests();
  });

  it('creates a mobile-uuid id and reuses it', () => {
    const storage = createTestStorage();
    const a = getOrCreateClientId(storage);
    const b = getOrCreateClientId(storage);
    expect(a).toBe(b);
    expect(a.startsWith(CLIENT_ID_PREFIX)).toBe(true);
    expect(a.toLowerCase()).not.toBe('default');
    expect(storage.getItem(CLIENT_ID_STORAGE_KEY)).toBe(a);
  });

  it('reuses stored id after cache reset', () => {
    const storage = createTestStorage({
      [CLIENT_ID_STORAGE_KEY]: 'mobile-fixed-id-123',
    });
    resetClientIdCacheForTests();
    expect(getOrCreateClientId(storage)).toBe('mobile-fixed-id-123');
    expect(peekClientId(storage)).toBe('mobile-fixed-id-123');
  });

  it('replaces forbidden stored default', () => {
    const storage = createTestStorage({
      [CLIENT_ID_STORAGE_KEY]: 'default',
    });
    const id = getOrCreateClientId(storage);
    expect(id).not.toBe('default');
    expect(id.startsWith(CLIENT_ID_PREFIX)).toBe(true);
  });
});
