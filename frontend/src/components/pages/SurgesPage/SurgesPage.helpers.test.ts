import { describe, expect, it } from 'vitest';
import { surgeHitsToRows } from './SurgesPage.helpers';

describe('surgeHitsToRows', () => {
  it('maps scan hits to table rows', () => {
    expect(
      surgeHitsToRows([
        {
          symbol: 'ETHUSDT',
          exchange: 'Binance',
          maxRatio: '4.2',
          hottest: '15m',
          grade: 'high',
          dominant: 'buy',
          summary: '15m is 4.2× typical',
        },
      ]),
    ).toEqual([
      {
        exchange: 'binance',
        symbol: 'ETHUSDT',
        maxRatio: '4.2',
        hottest: '15m',
        grade: 'high',
        dominant: 'buy',
        summary: '15m is 4.2× typical',
      },
    ]);
  });
});
