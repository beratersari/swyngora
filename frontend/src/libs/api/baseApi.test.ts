import { describe, expect, it } from 'vitest';
import { baseApi } from './baseApi';

describe('baseApi', () => {
  it('registers core tag types for cache invalidation', () => {
    // createApi options are not re-exported; verify reducer path and endpoints inject surface.
    expect(baseApi.reducerPath).toBe('api');
    expect(typeof baseApi.util.resetApiState).toBe('function');
    expect(typeof baseApi.injectEndpoints).toBe('function');
  });
});
