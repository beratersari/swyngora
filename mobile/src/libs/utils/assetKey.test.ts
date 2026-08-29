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

  it('does not collapse wrapped or staked tickers onto a shorter prefix', () => {
    expect(toSupplyAsset('WBTC')).toBe('WBTC');
    expect(toSupplyAsset('WBTCUSDT')).toBe('WBTC');
    expect(toSupplyAsset('STETH')).toBe('STETH');
    expect(toSupplyAsset('WSTETH')).toBe('WSTETH');
    expect(toSupplyAsset('W')).toBe('W');
    expect(toSupplyAsset('ETHBTC')).toBe('ETH');
  });
});
