import { describe, expect, it } from 'vitest';
import {
  normalizeSymbolRef,
  parseRealtimeMessage,
  realtimeWsUrl,
  reconnectDelayMs,
  symbolKey,
  uniqueSymbolRefs,
} from './helpers';
import { REALTIME_RECONNECT_MAX_MS, REALTIME_RECONNECT_MIN_MS } from './constants';

describe('realtime helpers', () => {
  it('normalizes and dedupes symbols', () => {
    expect(normalizeSymbolRef({ exchange: 'BINANCE', symbol: 'btcusdt' })).toEqual({
      exchange: 'binance',
      symbol: 'BTCUSDT',
    });
    expect(symbolKey({ exchange: 'binance', symbol: 'ethusdt' })).toBe('binance:ETHUSDT');
    expect(
      uniqueSymbolRefs([
        { exchange: 'binance', symbol: 'BTCUSDT' },
        { exchange: 'BINANCE', symbol: 'btcusdt' },
        { exchange: '', symbol: 'ETH' },
      ]),
    ).toEqual([{ exchange: 'binance', symbol: 'BTCUSDT' }]);
  });

  it('builds same-origin and absolute WS urls', () => {
    expect(realtimeWsUrl('', 'abc', 'localhost:5174', 'http:')).toBe(
      'ws://localhost:5174/api/v1/ws?clientId=abc',
    );
    expect(realtimeWsUrl('http://127.0.0.1:8080', 'c1')).toBe(
      'ws://127.0.0.1:8080/api/v1/ws?clientId=c1',
    );
    expect(realtimeWsUrl('https://api.example.com/', 'c1')).toBe(
      'wss://api.example.com/api/v1/ws?clientId=c1',
    );
  });

  it('caps reconnect backoff', () => {
    expect(reconnectDelayMs(0)).toBe(REALTIME_RECONNECT_MIN_MS);
    expect(reconnectDelayMs(1)).toBe(1000);
    expect(reconnectDelayMs(20)).toBe(REALTIME_RECONNECT_MAX_MS);
  });

  it('parses typed messages only', () => {
    expect(parseRealtimeMessage({ type: 'price', lastPrice: '1' })?.type).toBe('price');
    expect(parseRealtimeMessage({ nope: true })).toBeNull();
    expect(parseRealtimeMessage(null)).toBeNull();
  });
});
