import { describe, expect, it } from 'vitest';
import { exampleScaleOut, targetShareSum, targetWire, targetsComplete } from './targets';

describe('take-profit targets', () => {
  it('sends one absolute price as the single take-profit', () => {
    expect(
      targetWire([{ percent: 100, offsetPercent: null, price: 120 }]),
    ).toEqual({ takeProfitPrice: 120 });
  });

  it('sends the 25 / 25 / 50 scale-out as gains', () => {
    const rows = exampleScaleOut();
    expect(targetShareSum(rows)).toBe(100);
    expect(targetsComplete(rows)).toBe(true);
    expect(targetWire(rows)).toEqual({
      takeProfitTargets: [
        { percent: 25, offsetPercent: 5 },
        { percent: 25, offsetPercent: 10 },
        { percent: 50, offsetPercent: 15 },
      ],
    });
  });

  it('waits until the shares add up to 100 and each row has one price', () => {
    expect(targetsComplete([{ percent: 25, offsetPercent: 5, price: null }])).toBe(false);
    expect(
      targetsComplete([
        { percent: 50, offsetPercent: 5, price: 110 },
        { percent: 50, offsetPercent: null, price: null },
      ]),
    ).toBe(false);
  });
});
