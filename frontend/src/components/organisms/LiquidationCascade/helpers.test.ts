import { describe, expect, it } from 'vitest';
import { formatRatio, gradeTone, orderedWindows, sideTone } from './helpers';

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
});
