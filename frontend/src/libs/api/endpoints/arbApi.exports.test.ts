import { describe, expect, it } from 'vitest';
import { useGetPriceDiffOpportunityQuery } from '@/libs/api';
import { useGetPriceDiffOpportunityQuery as fromModule } from './arbApi';

describe('arbApi named exports', () => {
  it('exports useGetPriceDiffOpportunityQuery for the API barrel (native ESM)', () => {
    expect(typeof fromModule).toBe('function');
    expect(fromModule).toBe(useGetPriceDiffOpportunityQuery);
  });
});
