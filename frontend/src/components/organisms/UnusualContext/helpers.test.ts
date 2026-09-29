import { describe, expect, it } from 'vitest';
import { formatUnusualValue, unusualSentence } from './helpers';
import type { UnusualReadingView, UnusualTranslate } from './UnusualContext.types';

function spyT(): { t: UnusualTranslate; keys: string[] } {
  const keys: string[] = [];
  const t: UnusualTranslate = (key) => {
    keys.push(key);
    return key;
  };
  return { t, keys };
}

describe('unusual sentences', () => {
  it('picks print size, no book, no sample, few samples, and the funding line', () => {
    const cases: { reading: UnusualReadingView; key: string }[] = [
      {
        reading: { metric: 'print_size', reason: 'no_stored_history', current: 0 },
        key: 'tape.unusual.printSize',
      },
      {
        reading: { metric: 'spread', reason: 'no_book', current: 0 },
        key: 'tape.unusual.noBook',
      },
      {
        reading: { metric: 'liquidations', reason: 'no_sample', current: 0 },
        key: 'tape.unusual.noSample',
      },
      {
        reading: {
          metric: 'open_interest',
          reason: 'few_samples',
          band: 'unknown',
          samples: 4,
          minSamples: 20,
          spanHours: 10,
        },
        key: 'tape.unusual.few',
      },
      {
        reading: {
          metric: 'funding',
          band: 'high',
          label: 'predicted',
          current: 0.0003,
          percentile: 93,
          samples: 86,
          spanHours: 29 * 24,
          unit: 'rate',
        },
        key: 'tape.unusual.line.funding',
      },
    ];
    for (const item of cases) {
      const { t, keys } = spyT();
      expect(unusualSentence(item.reading, t)).toBe(item.key);
      expect(keys.at(-1)).toBe(item.key);
    }
  });

  it('formats rate, percent change, notional, signed flow, and spread', () => {
    expect(formatUnusualValue({ unit: 'rate', current: 0.0001 })).toBe('0.0100%');
    expect(formatUnusualValue({ unit: 'pct_change', current: 4.2 })).toBe('+4.20%');
    expect(formatUnusualValue({ unit: 'pct_change', current: -1.5 })).toBe('-1.50%');
    expect(formatUnusualValue({ unit: 'notional', current: 1_200_000 })).toBe('$1.20M');
    expect(formatUnusualValue({ unit: 'signed_notional', current: -400_000 })).toBe('-$400.00K');
    expect(formatUnusualValue({ unit: 'spread_pct', current: 0.0123 })).toBe('0.0123%');
    expect(
      formatUnusualValue({ unit: 'notional', reason: 'no_sample', current: 0 }),
    ).toBeNull();
  });
});
