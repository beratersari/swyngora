import { beforeEach, describe, expect, it } from 'vitest';
import {
  MARKETS_RETURN_STORAGE_KEY,
  marketsBackPath,
  rememberMarketsReturnPath,
} from './marketsReturn';

describe('marketsReturn', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('stores and restores full query string', () => {
    rememberMarketsReturnPath('exchange=coinbase&quote=USD&tag=defi');
    expect(sessionStorage.getItem(MARKETS_RETURN_STORAGE_KEY)).toBe(
      '?exchange=coinbase&quote=USD&tag=defi',
    );
    expect(marketsBackPath('binance')).toBe('/markets?exchange=coinbase&quote=USD&tag=defi');
  });

  it('empty search restores bare /markets', () => {
    rememberMarketsReturnPath('');
    expect(marketsBackPath('coinbase')).toBe('/markets');
  });
});
