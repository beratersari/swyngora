import { describe, expect, it } from 'vitest';
import { parseOptionsAsset } from './helpers';

describe('parseOptionsAsset', () => {
  it('accepts BTC, ETH, and pair aliases', () => {
    expect(parseOptionsAsset('ethusdt')).toBe('ETH');
    expect(parseOptionsAsset('BTCUSDT')).toBe('BTC');
    expect(parseOptionsAsset('nope')).toBe('BTC');
  });
});
