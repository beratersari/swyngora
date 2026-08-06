import { describe, expect, it } from 'vitest';
import { toRsiLineData } from './IndicatorChartHost.helpers';

describe('toRsiLineData', () => {
  it('returns empty for empty input', () => {
    expect(toRsiLineData([])).toEqual([]);
  });

  it('sorts by time and drops non-finite', () => {
    expect(
      toRsiLineData([
        { time: 3, value: 50 },
        { time: 1, value: 40 },
        { time: 2, value: Number.NaN },
        { time: Number.NaN, value: 10 },
      ]),
    ).toEqual([
      { time: 1, value: 40 },
      { time: 3, value: 50 },
    ]);
  });

  it('dedupes same time keeping last value', () => {
    expect(
      toRsiLineData([
        { time: 1, value: 10 },
        { time: 1, value: 20 },
      ]),
    ).toEqual([{ time: 1, value: 20 }]);
  });
});
