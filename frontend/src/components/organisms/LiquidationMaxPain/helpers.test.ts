import { describe, expect, it } from 'vitest';
import { pocketLevels, sideTone, venueLabel } from './helpers';

describe('LiquidationMaxPain helpers', () => {
  it('maps side and venue labels', () => {
    expect(sideTone('short')).toBe('up');
    expect(sideTone('long')).toBe('down');
    expect(venueLabel('binance')).toBe('Binance');
  });

  it('drops the primary pocket from the extra list', () => {
    const extra = pocketLevels(
      { price: '110', notional: '5000' },
      [
        { price: '110', notional: '5000' },
        { price: '115', notional: '800' },
      ],
    );
    expect(extra).toHaveLength(1);
    expect(extra[0]?.price).toBe('115');
  });
});
