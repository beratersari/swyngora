import { describe, expect, it } from 'vitest';
import { env } from './env';

describe('env', () => {
  it('exposes apiBaseUrl as a string (may be empty for same-origin)', () => {
    expect(typeof env.apiBaseUrl).toBe('string');
    if (env.apiBaseUrl.length > 0) {
      expect(env.apiBaseUrl.endsWith('/')).toBe(false);
    }
  });

  it('exposes a non-empty apiBaseUrlLabel for display', () => {
    expect(env.apiBaseUrlLabel.length).toBeGreaterThan(0);
  });
});
