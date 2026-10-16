import { describe, expect, it } from 'vitest';
import { situationChangeMarkers } from './situationMarks';

const candle = Math.floor(Date.parse('2024-01-01T00:00:00Z') / 1000);
const candles = [{ time: candle }];
const hour = 3600;

function label(clause: string, kind: string): string {
  return `${clause}:${kind}`;
}

describe('situationChangeMarkers', () => {
  it('joins clauses on one candle in card order and keeps the later kind', () => {
    const marks = situationChangeMarkers(
      [
        { clause: 'funding', at: '2024-01-01T00:00:00Z', kind: 'long' },
        { clause: 'funding', at: '2024-01-01T00:20:00Z', kind: 'short' },
        { clause: 'openInterest', at: '2024-01-01T00:10:00Z', kind: 'opened' },
        { clause: 'liquidation', at: '2024-01-01T00:00:00Z', kind: 'long' },
      ],
      candles,
      hour * 1.5,
      label,
    );
    expect(marks).toHaveLength(1);
    expect(marks[0]?.time).toBe(candle);
    expect(marks[0]?.shape).toBe('circle');
    expect(marks[0]?.text).toBe('openInterest:opened · funding:short');
  });

  it('draws the latest change of a clause and leaves the earlier one off', () => {
    const earlier = candle - hour;
    const marks = situationChangeMarkers(
      [
        { clause: 'funding', at: new Date(earlier * 1000).toISOString(), kind: 'long' },
        { clause: 'funding', at: new Date(candle * 1000).toISOString(), kind: 'short' },
      ],
      [{ time: earlier }, { time: candle }],
      hour * 1.5,
      label,
    );
    expect(marks).toHaveLength(1);
    expect(marks[0]?.time).toBe(candle);
    expect(marks[0]?.text).toBe('funding:short');
  });

  it('drops a change the loaded candles do not reach', () => {
    const marks = situationChangeMarkers(
      [{ clause: 'size', at: '2024-01-03T00:00:00Z', kind: 'flat' }],
      candles,
      hour * 1.5,
      label,
    );
    expect(marks).toEqual([]);
  });

  it('drops an unknown kind', () => {
    const marks = situationChangeMarkers(
      [{ clause: 'prints', at: '2024-01-01T00:00:00Z', kind: 'missing' }],
      candles,
      hour * 1.5,
      label,
    );
    expect(marks).toEqual([]);
  });
});
