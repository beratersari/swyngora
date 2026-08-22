import { describe, expect, it } from 'vitest';
import {
  livePumpEventsForPair,
  mergeChartMarkers,
  mergePumpEvents,
  pumpEventsToChartMarkers,
  scannerResultsToChartMarkers,
} from './CoinDetailPage.helpers';

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

describe('livePumpEventsForPair', () => {
  const btcEvent = { openTime: '2024-06-01T12:00:00Z', returnPct: 12.5 };

  it('keeps events for the current pair', () => {
    expect(
      livePumpEventsForPair(
        { exchange: 'binance', symbol: 'BTCUSDT', events: [btcEvent] },
        'binance',
        'BTCUSDT',
      ),
    ).toEqual([btcEvent]);
  });

  it('drops a previous coin payload so markers cannot snap onto the new series', () => {
    expect(
      livePumpEventsForPair(
        { exchange: 'binance', symbol: 'BTCUSDT', events: [btcEvent] },
        'binance',
        'ETHUSDT',
      ),
    ).toEqual([]);
  });

  it('returns empty when RTK currentData is missing (arg change)', () => {
    expect(livePumpEventsForPair(undefined, 'binance', 'ETHUSDT')).toEqual([]);
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

describe('scannerResultsToChartMarkers', () => {
  it('maps matching venue hits and stacks factors on one bar', () => {
    const out = scannerResultsToChartMarkers(
      [
        {
          id: '1',
          ruleId: 'a',
          exchange: 'binance',
          symbol: 'BTCUSDT',
          ruleType: 'rsi',
          interval: '4h',
          marketDataKey: '2026-08-01T12:00:00Z',
          matchedAt: '2026-08-01T12:01:00Z',
          summary: 'rsi',
        },
        {
          id: '2',
          ruleId: 'b',
          exchange: 'binance',
          symbol: 'BTCUSDT',
          ruleType: 'ma_crossover',
          interval: '4h',
          marketDataKey: '2026-08-01T12:00:00Z',
          matchedAt: '2026-08-01T12:01:00Z',
          summary: 'ema',
        },
        {
          id: '3',
          ruleId: 'c',
          exchange: 'bybit',
          symbol: 'BTCUSDT',
          ruleType: 'volume_increase',
          interval: '4h',
          marketDataKey: '2026-08-01T12:00:00Z',
          matchedAt: '2026-08-01T12:01:00Z',
          summary: 'vol',
        },
      ],
      'binance',
      'BTCUSDT',
    );
    expect(out).toHaveLength(1);
    expect(out[0]?.shape).toBe('circle');
    expect(out[0]?.text).toContain('RSI');
    expect(out[0]?.text).toContain('EMA');
  });
});

describe('mergeChartMarkers', () => {
  it('appends signal text onto a pump arrow on the same bar', () => {
    const out = mergeChartMarkers(
      [{ time: 100, position: 'belowBar', color: '#0f0', shape: 'arrowUp', text: '↑8.5' }],
      [{ time: 100, position: 'aboveBar', color: '#0ff', shape: 'circle', text: 'RSI' }],
    );
    expect(out).toHaveLength(1);
    expect(out[0]?.shape).toBe('arrowUp');
    expect(out[0]?.text).toBe('↑8.5 · RSI');
  });
});
