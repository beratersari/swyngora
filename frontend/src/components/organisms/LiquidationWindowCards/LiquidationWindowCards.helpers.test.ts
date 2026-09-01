import { describe, expect, it } from 'vitest';
import {
  isCardWindow,
  longShare,
  orderedCardWindows,
  parseNotional,
} from './LiquidationWindowCards.helpers';

describe('LiquidationWindowCards helpers', () => {
  it('orders the CoinGlass desk windows', () => {
    const ordered = orderedCardWindows([
      { window: '24h', totalNotional: '9' },
      { window: '1h', totalNotional: '1' },
    ]);
    expect(ordered.map((w) => w.window)).toEqual(['1h', '4h', '12h', '24h']);
    expect(ordered[0]?.totalNotional).toBe('1');
  });

  it('computes long share and guards empty totals', () => {
    expect(longShare({ longNotional: '80', shortNotional: '20' })).toBeCloseTo(0.8);
    expect(longShare({ longNotional: '0', shortNotional: '0' })).toBe(0.5);
    expect(parseNotional('nope')).toBe(0);
    expect(isCardWindow('12h')).toBe(true);
    expect(isCardWindow('5m')).toBe(false);
  });
});
