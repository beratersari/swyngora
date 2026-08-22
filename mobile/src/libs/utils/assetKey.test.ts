import { describe, expect, it } from 'vitest';
import { toSupplyAsset } from './assetKey';

describe('toSupplyAsset', () => {
  it('normalizes hyphen, fiat, and stable pair forms', () => {
    expect(toSupplyAsset('BTC-USD')).toBe('BTC');
    expect(toSupplyAsset('ETHTRY')).toBe('ETH');
    expect(toSupplyAsset('BTCUSDT')).toBe('BTC');
    expect(toSupplyAsset('RLUSD')).toBe('RLUSD');
    expect(toSupplyAsset('')).toBe('');
  });
});
