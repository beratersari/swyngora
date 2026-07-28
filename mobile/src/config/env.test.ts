import { describe, it, expect } from 'vitest';
import { env } from './env';

describe('env', () => {
  it('exposes apiBaseUrl string', () => {
    expect(typeof env.apiBaseUrl).toBe('string');
  });

  it('exposes a non-empty label', () => {
    expect(env.apiBaseUrlLabel.length).toBeGreaterThan(0);
  });
});
