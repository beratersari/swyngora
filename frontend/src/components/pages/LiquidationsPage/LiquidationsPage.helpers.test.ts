import { describe, expect, it } from 'vitest';
import {
  hasLiquidationTotals,
  parseLiqCascadeSymbol,
  parseLiqChartRange,
  parseLiqChartSymbol,
  parseLiqExchange,
  parseLiqSymbol,
  parseLiqHuntPanel,
  parseLiqView,
  parseLiqWindow,
} from './LiquidationsPage.helpers';

describe('LiquidationsPage helpers', () => {
  it('parses view, window, venue, and symbol', () => {
    expect(parseLiqView('heatmap')).toBe('heatmap');
    expect(parseLiqView('open')).toBe('open');
    expect(parseLiqView('chart')).toBe('chart');
    expect(parseLiqView('cascade')).toBe('cascade');
    expect(parseLiqView('hunt')).toBe('hunt');
    expect(parseLiqView('max-pain')).toBe('max-pain');
    expect(parseLiqHuntPanel('path')).toBe('path');
    expect(parseLiqHuntPanel(null)).toBe('compare');
    expect(parseLiqView(null)).toBe('heatmap');
    expect(parseLiqView('overview')).toBe('overview');
    expect(parseLiqCascadeSymbol(null)).toBe('all');
    expect(parseLiqCascadeSymbol('eth')).toBe('ETHUSDT');
    expect(parseLiqChartSymbol('all')).toBe('all');
    expect(parseLiqChartSymbol('eth')).toBe('ETHUSDT');
    expect(parseLiqChartRange('30d')).toBe('30d');
    expect(parseLiqChartRange('90d')).toBe('24h');
    expect(parseLiqChartRange(null)).toBe('24h');
    expect(parseLiqWindow('12h')).toBe('12h');
    expect(parseLiqWindow('5m')).toBe('24h');
    expect(parseLiqExchange('bybit')).toBe('bybit');
    expect(parseLiqExchange('okx')).toBe('okx');
    expect(parseLiqExchange('coinbase')).toBe('all');
    expect(parseLiqSymbol('btc-usd')).toBe('BTCUSDT');
    expect(parseLiqSymbol('')).toBe('BTCUSDT');
  });

  it('keeps stored overview totals when a venue is not definite', () => {
    expect(hasLiquidationTotals(undefined)).toBe(false);
    expect(hasLiquidationTotals({ coins: [], windows: [{ totalNotional: '0', count: 0 }] })).toBe(false);
    expect(hasLiquidationTotals({ coins: [{ symbol: 'BTCUSDT' }], windows: [] })).toBe(true);
    expect(hasLiquidationTotals({ windows: [{ totalNotional: '12', count: 1 }] })).toBe(true);
  });
});
