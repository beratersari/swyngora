import { describe, expect, it } from 'vitest';
import { footprintRows, imbalanceSide, isStrongImbalance } from './helpers';

describe('flow footprint helpers', () => {
  it('marks a 3x buy row and ignores a balanced row', () => {
    expect(isStrongImbalance({ buyVolume: '90', sellVolume: '20', sharePct: '4' })).toBe(true);
    expect(imbalanceSide({ buyVolume: '90', sellVolume: '20', sharePct: '4' })).toBe('buy');
    expect(isStrongImbalance({ buyVolume: '10', sellVolume: '40', sharePct: '2' })).toBe(true);
    expect(imbalanceSide({ buyVolume: '10', sellVolume: '40', sharePct: '2' })).toBe('sell');
    expect(isStrongImbalance({ buyVolume: '55', sellVolume: '45', sharePct: '8' })).toBe(false);
    expect(isStrongImbalance({ buyVolume: '5', sellVolume: '0', sharePct: '0.2' })).toBe(false);
    expect(isStrongImbalance({ buyVolume: '0', sellVolume: '80', sharePct: '2' })).toBe(true);
  });

  it('keeps the ladder highest price first', () => {
    const rows = footprintRows([
      { price: '100', volume: '1', buyVolume: '1', sellVolume: '0' },
      { price: '120', volume: '2', buyVolume: '1', sellVolume: '1' },
      { price: '0', volume: '0' },
    ]);
    expect(rows.map((row) => row.price)).toEqual(['120', '100']);
  });
});
