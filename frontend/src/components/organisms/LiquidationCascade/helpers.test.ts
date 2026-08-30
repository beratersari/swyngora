import { describe, expect, it } from 'vitest';
import { formatClock, formatDurationSec, formatRatio, gradeTone, orderedWindows, sideTone } from './helpers';

describe('LiquidationCascade helpers', () => {
  it('formats ratios and grades', () => {
    expect(formatRatio(4.2)).toBe('4.2×');
    expect(formatRatio(12)).toBe('12×');
    expect(formatRatio(0)).toBe('—');
    expect(gradeTone('cascade')).toBe('cascade');
    expect(gradeTone('nope')).toBe('quiet');
    expect(sideTone('long')).toBe('long');
    expect(sideTone(undefined)).toBe('none');
  });

  it('fills missing burst windows', () => {
    const rows = orderedWindows([{ window: '5m', maxRatio: 3 }]);
    expect(rows.map((r) => r.window)).toEqual(['1m', '5m', '15m']);
    expect(rows[1]?.maxRatio).toBe(3);
  });

  it('formats duration and UTC clocks', () => {
    expect(formatDurationSec(45)).toBe('45s');
    expect(formatDurationSec(600)).toBe('10m');
    expect(formatDurationSec(3600)).toBe('1h');
    expect(formatDurationSec(5400)).toBe('1h 30m');
    expect(formatClock('2026-08-30T15:21:00.000Z')).toBe('15:21');
  });
});
