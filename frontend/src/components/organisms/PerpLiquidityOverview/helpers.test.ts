import { describe, expect, it } from 'vitest';
import { bandLeaderLabel, bandShareLabel, bidSharePct, depthHistoryLine, depthLine, depthPriceLine, distanceBands, driverNames, leaderCoins, mixLine, moveAfterLine, moveBookLine, moveDriverLine, moveDuringLine, moveVenueLine, shareLabel, sizeDriverLine } from './helpers';

describe('PerpLiquidityOverview helpers', () => {
  it('drops empty or errored coins then keeps the largest', () => {
    const got = leaderCoins({
      coins: [
        { symbol: 'SOLUSDT', error: 'no perpetual depth' },
        { symbol: 'BTCUSDT', base: 'BTC', total: '100', sharePct: '80' },
        { symbol: 'ETHUSDT', base: 'ETH', total: '20', sharePct: '20' },
      ],
    });
    expect(got.map((c) => c.base)).toEqual(['BTC', 'ETH']);
  });

  it('puts the needle on bid as a share of bid plus ask', () => {
    expect(bidSharePct({ bid: '2400000000', ask: '1800000000' })).toBeCloseTo(57.14, 1);
    expect(bidSharePct({ bid: '0', ask: '0' })).toBeNull();
  });

  it('formats a share percent', () => {
    expect(shareLabel('48')).toBe('48%');
    expect(shareLabel(undefined)).toBe('—');
  });

  it('formats bid share then to now', () => {
    expect(mixLine({ complete: true, pastBidPct: '55', bidPct: '65', changePct: '+10' })).toBe(
      '55% → 65% (+10)',
    );
    expect(driverNames({ drivers: [{ base: 'ETH', changePct: '+10' }, { base: 'SOL' }] })).toBe(
      'ETH, SOL',
    );
  });

  it('formats a book jump and the BTC path around it', () => {
    const move = {
      pastTotal: '400000000',
      total: '900000000',
      changePct: '+125',
      priceFrom: '100000',
      priceTo: '99000',
      priceChangePct: '-1',
      priceAfter: '99500',
      priceAfterPct: '+0.5',
    };
    expect(moveBookLine(move)).toBe('400.00M → 900.00M (+125%)');
    expect(moveDuringLine(move)).toBe('100.00K → 99.00K (-1%)');
    expect(moveAfterLine(move)).toBe('99.50K (+0.5%)');
    expect(moveDuringLine({})).toBe('');
    expect(moveDriverLine({ drivers: [{ base: 'ETH', change: '+500000000', sharePct: '+83' }] })).toBe(
      'ETH +500.00M (+83%)',
    );
    expect(
      moveVenueLine({
        venues: [
          { exchange: 'binance', changePct: '-40' },
          { exchange: 'bybit', changePct: '-1.7' },
          { exchange: 'okx', changePct: '+2.5' },
        ],
      }),
    ).toBe('Binance -40%, Bybit -1.7%, OKX +2.5%');
  });

  it('formats total book size then to now', () => {
    expect(
      depthLine({ complete: true, pastTotal: '400000000', total: '500000000', totalChangePct: '+25' }),
    ).toBe('400.00M → 500.00M (+25%)');
    expect(depthLine({ complete: false, pastTotal: '1', total: '2' })).toBe('');
  });

  it('keeps minute totals in time order for the depth chart', () => {
    const line = depthHistoryLine([
      { time: '2026-09-21T16:00:00Z', total: '500' },
      { time: '2026-09-21T14:00:00Z', total: '400' },
      { time: 'bad', total: '1' },
    ]);
    expect(line.map((p) => p.value)).toEqual([400, 500]);
    expect(line[0].time).toBeLessThan(line[1].time);
    expect(
      depthPriceLine([
        { time: '2026-09-21T14:00:00Z', price: '100' },
        { time: '2026-09-21T16:00:00Z', price: '90' },
      ]).map((p) => p.value),
    ).toEqual([100, 90]);
  });

  it('names the coins that moved the book size, largest first', () => {
    expect(
      sizeDriverLine({
        sizeDrivers: [
          { base: 'ETH', change: '+100000000' },
          { base: 'BTC', change: '+20000000' },
          { base: 'SOL', change: '-10000000' },
        ],
      }),
    ).toBe('ETH +100.00M, BTC +20.00M, SOL -10.00M');
  });

  it('labels how much of the 1% book sits inside a tighter band', () => {
    const bands = distanceBands({
      rangePct: '1',
      bands: [
        { rangePct: '0.1', total: '80000000', ofHeadlinePct: '8', leaders: ['ETH', 'SOL'] },
        { rangePct: '1', total: '1000000000', ofHeadlinePct: '100', leaders: ['BTC'] },
      ],
    });
    expect(bandShareLabel(bands[0], '1')).toBe('80.00M · 8%');
    expect(bandShareLabel(bands[1], '1')).toBe('1.00B');
    expect(bandLeaderLabel(bands[0])).toBe('ETH, SOL');
  });
});
