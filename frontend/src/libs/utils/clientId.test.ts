import { describe, expect, it, beforeEach } from 'vitest';
import { getOrCreateClientId } from './clientId';

describe('getOrCreateClientId', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('persists a generated id', () => {
    const a = getOrCreateClientId();
    const b = getOrCreateClientId();
    expect(a).toBeTruthy();
    expect(a).toBe(b);
  });
});
