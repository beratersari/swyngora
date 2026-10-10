import { describe, expect, it } from 'vitest';
import { formatTapeReading, tapeDefaults, tapeQueryFromDraft } from './helpers';

describe('tape backtest helpers', () => {
  it('formats a share and a liquidation multiple', () => {
    expect(formatTapeReading('cvd', 0.65)).toBe('65.0%');
    expect(formatTapeReading('book', 0.7)).toBe('70.0%');
    expect(formatTapeReading('liquidation', 4)).toBe('4.00×');
    expect(formatTapeReading('funding', 0.01)).toBe('0.010%');
    expect(formatTapeReading('oi', 5)).toBe('5.0%');
    expect(formatTapeReading('gap', 3)).toBe('+3.0 pp');
    expect(formatTapeReading('size', 0.4)).toBe('40.0%');
    expect(formatTapeReading('cvd', null)).toBe('—');
  });

  it('builds a seven-day query from the draft', () => {
    const now = Date.parse('2026-10-06T00:00:00.000Z');
    const query = tapeQueryFromDraft(
      {
        symbol: 'btcusdt',
        condition: 'liquidation',
        side: 'long',
        exchange: 'all',
        days: 7,
        threshold: 4,
      },
      now,
    );
    expect(query.symbol).toBe('BTCUSDT');
    expect(query.from).toBe('2026-09-29T00:00:00.000Z');
    expect(tapeDefaults('book').side).toBe('bid');
  });
});
