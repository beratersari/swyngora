import { describe, expect, it } from 'vitest';
import { createTestStorage } from './storage';

describe('createTestStorage', () => {
  it('stores and reads keys', () => {
    const s = createTestStorage();
    expect(s.getItem('a')).toBeNull();
    s.setItem('a', '1');
    expect(s.getItem('a')).toBe('1');
  });

  it('seeds initial values', () => {
    const s = createTestStorage({ k: 'v' });
    expect(s.getItem('k')).toBe('v');
  });

  it('removes keys', () => {
    const s = createTestStorage({ k: 'v' });
    s.removeItem('k');
    expect(s.getItem('k')).toBeNull();
  });
});
