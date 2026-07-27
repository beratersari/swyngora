import { describe, expect, it } from 'vitest';
import { mergePumpEvents, pumpEventsToChartMarkers } from './CoinDetailPage.helpers';

describe('pumpEventsToChartMarkers', () => {
  it('maps pump and dump events to chart markers', () => {
    const out = pumpEventsToChartMarkers(
      [
        { openTime: '2024-06-01T12:00:00Z', returnPct: 8.5 },
        { openTime: '2024-06-01T13:00:00Z', returnPct: -6.25 },
      ],
      5,
    );
    expect(out).toHaveLength(2);
    expect(out[0]?.shape).toBe('arrowUp');
    expect(out[0]?.text).toBe('↑8.5');
    expect(out[1]?.shape).toBe('arrowDown');
    expect(out[1]?.text).toBe('↓6.3');
    expect(out[0]?.time).toBe(Math.floor(Date.parse('2024-06-01T12:00:00Z') / 1000));
  });

  it('filters by absolute threshold and drops bad times', () => {
    const out = pumpEventsToChartMarkers(
      [
        { openTime: '2024-06-01T12:00:00Z', returnPct: 3 },
        { openTime: 'not-a-date', returnPct: 20 },
        { openTime: '2024-06-01T14:00:00Z', returnPct: -5 },
      ],
      5,
    );
    expect(out).toHaveLength(1);
    expect(out[0]?.shape).toBe('arrowDown');
  });

  it('returns empty for missing events', () => {
    expect(pumpEventsToChartMarkers(undefined, 5)).toEqual([]);
    expect(pumpEventsToChartMarkers([], 5)).toEqual([]);
  });
});

describe('mergePumpEvents', () => {
  it('merges live and history pages by openTime and keeps stronger move', () => {
    const live = [{ openTime: '2024-06-02T00:00:00Z', returnPct: 6 }];
    const hist = [
      { openTime: '2024-06-01T00:00:00Z', returnPct: -8 },
      { openTime: '2024-06-02T00:00:00Z', returnPct: 3 }, // weaker than live
    ];
    const out = mergePumpEvents(live, hist);
    expect(out).toHaveLength(2);
    expect(out[0]?.openTime).toBe('2024-06-01T00:00:00Z');
    expect(out[1]?.returnPct).toBe(6);
  });
});
