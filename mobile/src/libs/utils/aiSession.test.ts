import { describe, expect, it, beforeEach } from 'vitest';
import { createTestStorage } from './storage';
import {
  getOrCreateAiSessionId,
  peekAiSessionId,
  resetAiSessionCacheForTests,
  rotateAiSessionId,
} from './aiSession';

describe('aiSession', () => {
  beforeEach(() => {
    resetAiSessionCacheForTests();
  });

  it('creates a stable mobile-ai session id', () => {
    const storage = createTestStorage();
    const a = getOrCreateAiSessionId(storage);
    const b = getOrCreateAiSessionId(storage);
    expect(a).toBe(b);
    expect(a.startsWith('mobile-ai-')).toBe(true);
    expect(a.toLowerCase()).not.toBe('default');
    expect(a.toLowerCase()).not.toBe('http-default');
  });

  it('rejects forbidden stored ids', () => {
    const storage = createTestStorage({
      'swyngora.mobile.aiSessionId.v1': 'default',
    });
    const id = getOrCreateAiSessionId(storage);
    expect(id.startsWith('mobile-ai-')).toBe(true);
    expect(id).not.toBe('default');
  });

  it('rotates session id for new chat', () => {
    const storage = createTestStorage();
    const a = getOrCreateAiSessionId(storage);
    const b = rotateAiSessionId(storage);
    expect(b).not.toBe(a);
    expect(peekAiSessionId(storage)).toBe(b);
  });
});
