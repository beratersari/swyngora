import { describe, expect, it } from 'vitest';
import { nearestTime, snapMarkersToCandleTimes } from './CandleChartHost.markers';
import type { CandleChartMarker } from './CandleChartHost.types';
import type { ChartCandle } from '@/libs/utils';

function bar(time: number): ChartCandle {
  return { time, open: 1, high: 1, low: 1, close: 1 };
}

describe('nearestTime', () => {
  it('finds exact and nearest', () => {
    const t = [10, 20, 30, 40];
    expect(nearestTime(t, 20)).toBe(20);
    expect(nearestTime(t, 22)).toBe(20);
    expect(nearestTime(t, 28)).toBe(30);
    expect(nearestTime(t, 1)).toBe(10);
    expect(nearestTime(t, 100)).toBe(40);
  });
});

describe('snapMarkersToCandleTimes', () => {
  it('snaps off-grid marker times onto bars so LWC can draw them', () => {
    const candles = [100, 200, 300, 400].map(bar);
    const markers: CandleChartMarker[] = [
      {
        time: 205, // not a bar — would vanish on zoom without snap
        position: 'belowBar',
        color: '#0f0',
        shape: 'arrowUp',
      },
      {
        time: 300,
        position: 'aboveBar',
        color: '#f00',
        shape: 'arrowDown',
      },
    ];
    const snapped = snapMarkersToCandleTimes(markers, candles);
    expect(snapped.map((m) => m.time)).toEqual([200, 300]);
  });

  it('dedupes markers that snap to the same bar', () => {
    const candles = [100, 200].map(bar);
    const markers: CandleChartMarker[] = [
      { time: 195, position: 'belowBar', color: '#0f0', shape: 'arrowUp', text: 'a' },
      { time: 205, position: 'belowBar', color: '#0f0', shape: 'arrowUp', text: 'b' },
    ];
    const snapped = snapMarkersToCandleTimes(markers, candles);
    expect(snapped).toHaveLength(1);
    expect(snapped[0]!.time).toBe(200);
  });
});
