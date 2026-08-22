import { describe, expect, it } from 'vitest';
import { BROWSER_API_TOKEN_KEY, getBrowserApiToken, setBrowserApiToken } from './apiAuth';

describe('browser API token', () => {
  it('stores and clears a user-issued key', () => {
    setBrowserApiToken('swy_test');
    expect(getBrowserApiToken()).toBe('swy_test');
    expect(localStorage.getItem(BROWSER_API_TOKEN_KEY)).toBe('swy_test');
    setBrowserApiToken('');
    expect(getBrowserApiToken()).toBe('');
  });
});
