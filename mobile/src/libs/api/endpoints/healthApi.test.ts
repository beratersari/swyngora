import { describe, expect, it } from 'vitest';
import { healthApi } from './healthApi';

describe('healthApi', () => {
  it('registers getHealth endpoint', () => {
    expect(healthApi.endpoints.getHealth).toBeDefined();
    expect(typeof healthApi.endpoints.getHealth.initiate).toBe('function');
  });
});
