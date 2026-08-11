import { describe, expect, it } from 'vitest';
import { tapeCellLabel, toTickerTapeItem } from './TickerTape.helpers';

describe('TickerTape helpers', () => {
  it('maps a spot row to a tape item', () => {
    const item = toTickerTapeItem({
      exchange: 'binance',
      symbol: 'BTCUSDT',
      lastPrice: '67000',
      priceChangePercent: '1.5',
    });
    expect(item?.href).toBe('/markets/binance/BTCUSDT');
    expect(item?.changeValue).toBe(1.5);
    expect(item?.lastPrice).not.toBe('—');
    expect(tapeCellLabel(item!)).toContain('BTC/USDT');
  });

  it('returns null without exchange or symbol', () => {
    expect(toTickerTapeItem({ symbol: 'BTCUSDT' })).toBeNull();
  });
});
