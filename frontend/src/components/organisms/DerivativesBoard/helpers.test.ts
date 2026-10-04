import { describe, expect, it } from 'vitest';
import {
  coinChartPath,
  definiteText,
  formatRecordedSpan,
  fundingText,
  percentText,
  recordedSummary,
  signedAmount,
  sortBoardCoins,
} from './helpers';
import type { DerivativesCoin } from '@/libs/api';

describe('derivatives board helpers', () => {
  it('opens the binance chart for a combined row', () => {
    expect(coinChartPath('all', 'BTCUSDT')).toBe('/markets/binance/BTCUSDT');
    expect(coinChartPath('bybit', 'ETHUSDT')).toBe('/markets/bybit/ETHUSDT');
  });

  it('withholds a combined number that is not definite', () => {
    expect(definiteText(false, '100', (value) => value)).toBe('—');
    expect(definiteText(true, '100', (value) => value)).toBe('100');
  });

  it('signs a compact open-interest change', () => {
    expect(signedAmount('1500000000')).toBe('+1.50B');
    expect(signedAmount('-2500000')).toBe('-2.50M');
  });

  it('leaves funding blank until every venue is fresh', () => {
    const coin = { fundingDefinite: false, fundingRatePct: '+0.01' } as DerivativesCoin;
    expect(fundingText(coin)).toBe('—');
    expect(fundingText({ ...coin, fundingDefinite: true })).toBe('+0.01%');
  });

  it('shows the longest stored span for each series', () => {
    expect(
      recordedSummary(
        [
          { kind: 'futures_oi', exchange: 'binance', spanSec: 3600 },
          { kind: 'futures_oi', exchange: 'bybit', spanSec: 7200 },
          { kind: 'book', exchange: 'binance', spanSec: 86400 },
        ],
        (kind) => kind,
        formatRecordedSpan,
      ),
    ).toBe('futures_oi 2h · book 24h');
    expect(recordedSummary([], (kind) => kind, formatRecordedSpan)).toBe('—');
  });

  it('shows crowd long and taker buy share as percents', () => {
    expect(percentText(false, '55')).toBe('—');
    expect(percentText(true, '55')).toBe('55%');
  });

  it('sorts a column and keeps a blank cell at the bottom', () => {
    const coins = [
      { symbol: 'ETHUSDT', base: 'ETH', longPct: '40', longDefinite: true },
      { symbol: 'BTCUSDT', base: 'BTC', longDefinite: false, longPct: '80' },
      { symbol: 'SOLUSDT', base: 'SOL', longPct: '70', longDefinite: true },
    ] as DerivativesCoin[];
    expect(sortBoardCoins(coins, { key: 'longPct', dir: 'desc' }).map((coin) => coin.base)).toEqual([
      'SOL',
      'ETH',
      'BTC',
    ]);
    expect(sortBoardCoins(coins, { key: 'base', dir: 'asc' }).map((coin) => coin.base)).toEqual([
      'BTC',
      'ETH',
      'SOL',
    ]);
  });

  it('sorts day CVD and keeps a blank tape at the bottom', () => {
    const coins = [
      { symbol: 'ETHUSDT', base: 'ETH', cvdDay: '-20', cvdDayDefinite: true },
      { symbol: 'BTCUSDT', base: 'BTC', cvdDay: '80', cvdDayDefinite: false },
      { symbol: 'SOLUSDT', base: 'SOL', cvdDay: '40', cvdDayDefinite: true },
    ] as DerivativesCoin[];
    expect(sortBoardCoins(coins, { key: 'cvdDay', dir: 'desc' }).map((coin) => coin.base)).toEqual([
      'SOL',
      'ETH',
      'BTC',
    ]);
  });
});
