import { describe, expect, it } from 'vitest';
import {
  pumpEventsToChartMarkers,
  pumpEventsToMarginLines,
} from './pumpChart';
import type { PumpEvent } from '@/libs/api';

const upEvent: PumpEvent = {
  openTime: '2026-07-26T06:45:00Z',
  startPrice: 1,
  endPrice: 1.1,
  returnPct: 10,
  high: 1.12,
  low: 0.99,
};

const downEvent: PumpEvent = {
  openTime: '2026-07-26T08:00:00Z',
  startPrice: 2,
  endPrice: 1.8,
  returnPct: -10,
  high: 2.05,
  low: 1.75,
};

describe('pumpEventsToChartMarkers', () => {
  it('maps up/down markers sorted by time', () => {
    const markers = pumpEventsToChartMarkers([downEvent, upEvent]);
    expect(markers).toHaveLength(2);
    expect(markers[0].shape).toBe('arrowUp');
    expect(markers[0].text).toBe('+10.00%');
    expect(markers[1].shape).toBe('arrowDown');
    expect(markers[0].time).toBeLessThan(markers[1].time);
  });

  it('skips invalid openTime', () => {
    expect(pumpEventsToChartMarkers([{ openTime: 'nope', returnPct: 1 }])).toEqual([]);
  });
});

describe('pumpEventsToMarginLines', () => {
  it('emits high/low and distinct start/end lines', () => {
    const lines = pumpEventsToMarginLines([upEvent]);
    const prices = lines.map((l) => l.price);
    expect(prices).toContain(1.12);
    expect(prices).toContain(0.99);
    expect(prices).toContain(1);
    expect(prices).toContain(1.1);
  });

  it('returns empty for no events', () => {
    expect(pumpEventsToMarginLines([])).toEqual([]);
    expect(pumpEventsToMarginLines(undefined)).toEqual([]);
  });
});
