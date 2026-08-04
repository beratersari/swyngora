import { beforeEach, describe, expect, it } from 'vitest';
import { getOrCreateAiSessionId, resetAiSessionId } from './aiSessionId';

describe('aiSessionId', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('creates and reuses a stable session id', () => {
    const a = getOrCreateAiSessionId();
    const b = getOrCreateAiSessionId();
    expect(a).toMatch(/^web-ai-/);
    expect(b).toBe(a);
  });

  it('resetAiSessionId issues a new id', () => {
    const a = getOrCreateAiSessionId();
    const b = resetAiSessionId();
    expect(b).toMatch(/^web-ai-/);
    expect(b).not.toBe(a);
    expect(getOrCreateAiSessionId()).toBe(b);
  });
});
