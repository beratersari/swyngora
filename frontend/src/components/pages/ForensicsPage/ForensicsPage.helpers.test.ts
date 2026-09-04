import { describe, expect, it } from 'vitest';
import {
  datetimeLocalToUtc,
  defaultForensicsAt,
  parseForensicsAt,
  parseForensicsDuring,
  parseForensicsExchange,
  parseForensicsSymbol,
  parseForensicsWindow,
  utcToDatetimeLocal,
} from './ForensicsPage.helpers';

describe('ForensicsPage helpers', () => {
  it('defaults and normalizes symbol', () => {
    expect(parseForensicsSymbol(null)).toBe('BTCUSDT');
    expect(parseForensicsSymbol(' ethusdt ')).toBe('ETHUSDT');
  });

  it('parses exchange window and during', () => {
    expect(parseForensicsExchange('bybit')).toBe('bybit');
    expect(parseForensicsExchange('okx')).toBe('all');
    expect(parseForensicsWindow('4h')).toBe('4h');
    expect(parseForensicsWindow('9h')).toBe('1h');
    expect(parseForensicsDuring('5m')).toBe('5m');
    expect(parseForensicsDuring('2h')).toBe('15m');
  });

  it('defaults at to one hour ago rounded to 15 minutes', () => {
    const now = Date.UTC(2026, 8, 4, 14, 22, 40);
    expect(defaultForensicsAt(now)).toBe('2026-09-04T13:15:00.000Z');
    expect(parseForensicsAt(null, now)).toBe('2026-09-04T13:15:00.000Z');
    expect(parseForensicsAt('2026-08-20T14:00:00Z', now)).toBe('2026-08-20T14:00:00.000Z');
    expect(parseForensicsAt('not-a-date', now)).toBe('2026-09-04T13:15:00.000Z');
  });

  it('round-trips datetime-local through UTC', () => {
    const iso = '2026-08-20T14:00:00.000Z';
    const local = utcToDatetimeLocal(iso);
    expect(local).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
    expect(datetimeLocalToUtc(local)).toBe(iso);
  });
});
