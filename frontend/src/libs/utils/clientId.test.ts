import { describe, expect, it, beforeEach, vi } from 'vitest';
import { getOrCreateClientId } from './clientId';

describe('getOrCreateClientId', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('persists a generated id', () => {
    const a = getOrCreateClientId();
    const b = getOrCreateClientId();
    expect(a).toBeTruthy();
    expect(a).not.toBe('anonymous');
    expect(a).toBe(b);
  });

  it('fails closed when storage throws', () => {
    const spy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    expect(() => getOrCreateClientId()).toThrow(/storage unavailable/);
    spy.mockRestore();
  });
});
