import { describe, expect, it } from 'vitest';
import {
  change24hMissingTitle,
  coinChartPath,
  definiteText,
  formatRecordedSpan,
  fundingText,
  orderCountText,
  cvdSplitText,
  percentText,
  recordedSummary,
  signedAmount,
  sortBoardCoins,
  unusualText,
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

  it('names the venues with no 24h open-interest sample', () => {
    expect(change24hMissingTitle(['okx'], (id) => id.toUpperCase())).toBe('OKX');
    expect(change24hMissingTitle(['binance', ' bybit '], (id) => id)).toBe('binance, bybit');
    expect(change24hMissingTitle(undefined, (id) => id)).toBe('');
    expect(change24hMissingTitle([], (id) => id)).toBe('');
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

  it('shows the 15-minute order-count ratio and the named unusual venue', () => {
    const counted = { orderDefinite: true, orderRatio: '+0.40' } as DerivativesCoin;
    expect(orderCountText(counted, 'Buy')).toBe('+0.40 Buy');
    expect(orderCountText({ orderDefinite: false, orderRatio: '+0.40' }, 'Buy')).toBe('—');
    const standout = {
      unusualDefinite: true,
      unusualMetric: 'funding',
      unusualBand: 'extreme_high',
      unusualVenue: 'binance',
    } as DerivativesCoin;
    expect(unusualText(standout, 'Funding', 'Extremely high', 'Binance')).toBe('Funding · Extremely high · Binance');
    expect(unusualText({ unusualDefinite: false }, 'Funding', 'Extremely high', 'Binance')).toBe('—');
  });

  it('sorts order count and unusual and keeps a blank cell at the bottom', () => {
    const coins = [
      { symbol: 'ETHUSDT', base: 'ETH', orderRatio: '-0.20', orderDefinite: true, unusualDistance: '10', unusualDefinite: true },
      { symbol: 'BTCUSDT', base: 'BTC', orderRatio: '+0.80', orderDefinite: false, unusualDistance: '49', unusualDefinite: false },
      { symbol: 'SOLUSDT', base: 'SOL', orderRatio: '+0.40', orderDefinite: true, unusualDistance: '40', unusualDefinite: true },
    ] as DerivativesCoin[];
    expect(sortBoardCoins(coins, { key: 'orderCount', dir: 'desc' }).map((coin) => coin.base)).toEqual([
      'SOL',
      'ETH',
      'BTC',
    ]);
    expect(sortBoardCoins(coins, { key: 'unusual', dir: 'desc' }).map((coin) => coin.base)).toEqual([
      'SOL',
      'ETH',
      'BTC',
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

  it('sorts a price-versus-CVD run by signed duration and keeps a blank tape last', () => {
    const coins = [
      { symbol: 'ETHUSDT', base: 'ETH', cvdSplit: 'price_down_cvd_up', cvdSplitSeconds: '-600', cvdSplitDefinite: true },
      { symbol: 'BTCUSDT', base: 'BTC', cvdSplit: 'price_up_cvd_down', cvdSplitSeconds: '2100', cvdSplitDefinite: false },
      { symbol: 'SOLUSDT', base: 'SOL', cvdSplit: 'agrees', cvdSplitSeconds: '0', cvdSplitDefinite: true },
      { symbol: 'XRPUSDT', base: 'XRP', cvdSplit: 'price_up_cvd_down', cvdSplitSeconds: '1200', cvdSplitDefinite: true },
    ] as DerivativesCoin[];
    expect(sortBoardCoins(coins, { key: 'cvdSplit', dir: 'desc' }).map((coin) => coin.base)).toEqual([
      'XRP',
      'SOL',
      'ETH',
      'BTC',
    ]);
    const label = { up: 'Price up, CVD down', down: 'Price down, CVD up', agrees: 'In line' };
    expect(cvdSplitText(coins[3]!, label)).toBe('Price up, CVD down');
    expect(cvdSplitText({ ...coins[3]!, cvdSplitDuration: '20m' }, label)).toBe('Price up, CVD down · 20m');
    expect(cvdSplitText(coins[2]!, label)).toBe('In line');
    expect(cvdSplitText(coins[1]!, label)).toBe('—');
  });
});
