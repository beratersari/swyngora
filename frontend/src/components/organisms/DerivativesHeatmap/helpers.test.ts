import { describe, expect, it } from 'vitest';
import type { DerivativesCoin } from '@/libs/api';
import { heatFill, heatmapCells } from './helpers';

function coin(partial: DerivativesCoin): DerivativesCoin {
  return { symbol: 'BTCUSDT', base: 'BTC', ...partial };
}

describe('derivatives heatmap', () => {
  it('keeps an incomplete column blank and does not treat it as zero', () => {
    const cells = heatmapCells('all', [
      coin({ fundingRatePct: '+0.20', fundingDefinite: false }),
      coin({ symbol: 'ETHUSDT', base: 'ETH', fundingRatePct: '+0.01', fundingDefinite: true }),
    ], 'funding');
    expect(cells[0]?.label).toBe('ETH');
    expect(cells[0]?.tone).toBe('up');
    expect(cells[1]?.tone).toBe('blank');
    expect(cells[1]?.text).toBe('—');
    expect(cells[1]?.strength).toBe(0);
  });

  it('puts the larger distance from neutral first', () => {
    const cells = heatmapCells('all', [
      coin({ orderRatio: '+0.20', orderDefinite: true }),
      coin({ symbol: 'ETHUSDT', base: 'ETH', orderRatio: '-0.80', orderDefinite: true }),
      coin({ symbol: 'SOLUSDT', base: 'SOL', orderRatio: '0', orderDefinite: true }),
      coin({ symbol: 'XRPUSDT', base: 'XRP', orderRatio: '+1.00', orderDefinite: true }),
    ], 'orderCount');
    expect(cells.map((cell) => cell.label)).toEqual(['XRP', 'ETH', 'BTC', 'SOL']);
    expect(cells[0]?.tone).toBe('up');
    expect(cells[0]?.strength).toBe(1);
    expect(cells[1]?.tone).toBe('down');
    expect(cells[3]?.tone).toBe('flat');
  });

  it('colors unusual highs and lows from the band, not from a missing sign', () => {
    const cells = heatmapCells('bybit', [
      coin({ unusualDefinite: true, unusualDistance: '10', unusualBand: 'high' }),
      coin({ symbol: 'ETHUSDT', base: 'ETH', unusualDefinite: true, unusualDistance: '49', unusualBand: 'extreme_low' }),
    ], 'unusual');
    expect(cells[0]).toMatchObject({ label: 'ETH', text: '-49', tone: 'down', path: '/markets/bybit/ETHUSDT' });
    expect(cells[1]).toMatchObject({ label: 'BTC', text: '+10', tone: 'up' });
  });

  it('paints liquidation heat from a ratio of 1 and day CVD against the screen', () => {
    const heat = heatmapCells('all', [
      coin({ liqHeatDefinite: true, liqHeatRatio: '1' }),
      coin({ symbol: 'ETHUSDT', base: 'ETH', liqHeatDefinite: true, liqHeatRatio: '4' }),
    ], 'liqHeat');
    expect(heat[0]).toMatchObject({ label: 'ETH', tone: 'heat', strength: 1, text: '4×' });
    expect(heat[1]).toMatchObject({ label: 'BTC', tone: 'flat', strength: 0 });

    const cvd = heatmapCells('all', [
      coin({ cvdDayDefinite: true, cvdDay: '-100' }),
      coin({ symbol: 'ETHUSDT', base: 'ETH', cvdDayDefinite: true, cvdDay: '1000' }),
    ], 'cvdDay');
    expect(cvd[0]).toMatchObject({ label: 'ETH', tone: 'up', strength: 1 });
    expect(cvd[1]?.strength).toBeCloseTo(0.1);
  });

  it('leaves a blank cell on the bed color', () => {
    expect(heatFill('blank', 0)).toBe('#E9EEF5');
    expect(heatFill('up', 1)).toContain('rgba(22, 199, 132');
  });
});
