import { describe, it, expect } from 'vitest';
import {
  buildCrossExchangePlan,
  cheapestExchangeId,
  mapTickerToCrossExchangeRow,
  parseMarketSymbol,
  symbolCandidatesForExchange,
  type CrossExchangeRowModel,
} from './crossExchange';

describe('parseMarketSymbol', () => {
  it('parses compact USDT pairs', () => {
    expect(parseMarketSymbol('binance', 'BTCUSDT')).toEqual({
      base: 'BTC',
      quote: 'USDT',
    });
  });

  it('parses hyphenated Coinbase pairs', () => {
    expect(parseMarketSymbol('coinbase', 'eth-usd')).toEqual({
      base: 'ETH',
      quote: 'USD',
    });
  });

  it('handles empty', () => {
    expect(parseMarketSymbol('binance', '')).toBeNull();
  });
});

describe('symbolCandidatesForExchange', () => {
  it('lists coinbase hyphenated first', () => {
    expect(symbolCandidatesForExchange('BTC', 'coinbase')[0]).toBe('BTC-USD');
  });

  it('lists binance compact USDT first', () => {
    expect(symbolCandidatesForExchange('BTC', 'binance')[0]).toBe('BTCUSDT');
  });
});

describe('buildCrossExchangePlan', () => {
  it('keeps source symbol and maps others from base', () => {
    const plan = buildCrossExchangePlan('binance', 'BTCUSDT');
    const source = plan.find((p) => p.isSource);
    expect(source?.exchange).toBe('binance');
    expect(source?.candidates).toEqual(['BTCUSDT']);
    const cb = plan.find((p) => p.exchange === 'coinbase');
    expect(cb?.candidates[0]).toBe('BTC-USD');
    expect(cb?.isSource).toBe(false);
  });

  it('uses coinbase route as source', () => {
    const plan = buildCrossExchangePlan('coinbase', 'BTC-USD');
    expect(plan.find((p) => p.isSource)?.candidates).toEqual(['BTC-USD']);
    expect(plan.find((p) => p.exchange === 'binance')?.candidates[0]).toBe(
      'BTCUSDT',
    );
  });
});

describe('mapTickerToCrossExchangeRow', () => {
  it('maps ok ticker', () => {
    const plan = buildCrossExchangePlan('binance', 'BTCUSDT')[0]!;
    const row = mapTickerToCrossExchangeRow(plan, 'BTCUSDT', {
      symbol: 'BTCUSDT',
      lastPrice: '67000',
      priceChangePercent: '1.5',
      quoteVolume: '1500000000',
    }, { status: 'ok' });
    expect(row.status).toBe('ok');
    expect(row.changePercentLabel).toContain('1.50%');
    expect(row.quoteVolumeLabel).toContain('B');
  });

  it('maps unavailable without inventing prices', () => {
    const plan = buildCrossExchangePlan('binance', 'BTCUSDT')[1]!;
    const row = mapTickerToCrossExchangeRow(plan, 'BTC-USD', undefined, {
      status: 'unavailable',
    });
    expect(row.lastPriceLabel).toBe('—');
    expect(row.status).toBe('unavailable');
  });
});

describe('cheapestExchangeId', () => {
  it('picks lowest ok price', () => {
    const rows: CrossExchangeRowModel[] = [
      {
        id: 'a',
        exchange: 'binance',
        symbol: 'X',
        isSource: true,
        lastPriceLabel: '100',
        changePercentLabel: '+0%',
        changeTone: 'secondary',
        quoteVolumeLabel: '1',
        status: 'ok',
      },
      {
        id: 'b',
        exchange: 'coinbase',
        symbol: 'Y',
        isSource: false,
        lastPriceLabel: '90',
        changePercentLabel: '+0%',
        changeTone: 'secondary',
        quoteVolumeLabel: '1',
        status: 'ok',
      },
    ];
    expect(cheapestExchangeId(rows)).toBe('b');
  });
});
