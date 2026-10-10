import { describe, expect, it } from 'vitest';
import { clickSecFromForensicsAt, coinForensicsHref } from './helpers';

describe('coinForensicsHref', () => {
  it('keeps the forensics desk defaults for the clicked candle', () => {
    const href = coinForensicsHref('BTCUSDT', '2026-08-20T14:00:00.000Z');
    const params = new URLSearchParams(href.slice(href.indexOf('?') + 1));
    expect(href.startsWith('/forensics?')).toBe(true);
    expect(params.get('view')).toBe('event');
    expect(params.get('symbol')).toBe('BTCUSDT');
    expect(params.get('at')).toBe('2026-08-20T14:00:00.000Z');
    expect(params.get('exchange')).toBe('all');
    expect(params.get('window')).toBe('1h');
    expect(params.get('during')).toBe('15m');
  });
});

describe('clickSecFromForensicsAt', () => {
  it('reads a similar-move time as unix seconds', () => {
    expect(clickSecFromForensicsAt('2026-08-18T12:00:00.000Z')).toBe(
      Math.floor(Date.parse('2026-08-18T12:00:00.000Z') / 1000),
    );
  });

  it('leaves an unreadable time off the chart', () => {
    expect(clickSecFromForensicsAt('not-a-time')).toBeNull();
  });
});
