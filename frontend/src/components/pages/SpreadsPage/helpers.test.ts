import { describe, expect, it } from 'vitest';
import { hitsToRows } from './helpers';

describe('spread rows', () => {
  it('keeps the widest book and formats the ratio', () => {
    const rows = hitsToRows([
      { symbol: 'BTCUSDT', exchange: 'binance', currentPct: '0.08', typicalPct: '0.01', ratio: '8' },
      { symbol: '', exchange: 'bybit' },
    ]);
    expect(rows).toEqual([
      { exchange: 'binance', symbol: 'BTCUSDT', current: '0.08%', typical: '0.01%', ratio: '8×' },
    ]);
  });
});
