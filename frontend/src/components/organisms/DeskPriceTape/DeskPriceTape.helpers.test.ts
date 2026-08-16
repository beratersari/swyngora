import { describe, expect, it, beforeEach } from 'vitest';
import { DESK_TAPE_SOURCE_STORAGE_KEY } from './DeskPriceTape.constants';
import { isDeskTapeSource, loadDeskTapeSource, saveDeskTapeSource } from './DeskPriceTape.helpers';

describe('DeskPriceTape helpers', () => {
  beforeEach(() => {
    localStorage.removeItem(DESK_TAPE_SOURCE_STORAGE_KEY);
  });

  it('accepts the four tape sources', () => {
    expect(isDeskTapeSource('binance')).toBe(true);
    expect(isDeskTapeSource('coinbase')).toBe(true);
    expect(isDeskTapeSource('bist')).toBe(true);
    expect(isDeskTapeSource('watchlist')).toBe(true);
    expect(isDeskTapeSource('nasdaq')).toBe(false);
  });

  it('defaults to binance and persists a choice', () => {
    expect(loadDeskTapeSource()).toBe('binance');
    saveDeskTapeSource('bist');
    expect(loadDeskTapeSource()).toBe('bist');
  });
});
