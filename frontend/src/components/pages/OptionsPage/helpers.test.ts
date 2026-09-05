import { describe, expect, it } from 'vitest';
import { expiryRowKey } from '@/components/organisms/OptionsExpiryBars/helpers';
import { parseOptionsAsset, parseSelectedExpiry } from './helpers';

describe('parseOptionsAsset', () => {
  it('accepts BTC, ETH, and pair aliases', () => {
    expect(parseOptionsAsset('ethusdt')).toBe('ETH');
    expect(parseOptionsAsset('BTCUSDT')).toBe('BTC');
    expect(parseOptionsAsset('nope')).toBe('BTC');
  });
});

describe('parseSelectedExpiry', () => {
  const rows = [
    { expiry: '2026-03-27T08:00:00Z', label: '27 Mar 2026', daysToExpiry: 20 },
    { expiry: '2026-06-26T08:00:00Z', label: '26 Jun 2026', daysToExpiry: 111 },
  ];

  it('defaults to the nearest live expiry', () => {
    expect(parseSelectedExpiry(null, rows)).toBe('2026-03-27');
  });

  it('matches a query date or label', () => {
    expect(parseSelectedExpiry('2026-06-26', rows)).toBe('2026-06-26');
    expect(parseSelectedExpiry('26 Jun 2026', rows)).toBe('2026-06-26');
  });

  it('builds a date key from the expiry timestamp', () => {
    expect(expiryRowKey(rows[0])).toBe('2026-03-27');
  });
});
