import { describe, expect, it } from 'vitest';
import { env } from './env';

describe('env', () => {
  it('exposes apiBaseUrl, label, and clientId strings', () => {
    expect(typeof env.apiBaseUrl).toBe('string');
    expect(typeof env.apiBaseUrlLabel).toBe('string');
    expect(typeof env.clientId).toBe('string');
    expect(env.apiBaseUrlLabel.length).toBeGreaterThan(0);
    if (env.apiBaseUrl) {
      expect(env.apiBaseUrl.endsWith('/')).toBe(false);
    }
  });
});
