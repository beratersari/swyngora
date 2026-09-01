import { describe, expect, it } from 'vitest';
import {
  parseLiqCascadeSymbol,
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
    expect(parseLiqView('chart')).toBe('chart');
    expect(parseLiqView('cascade')).toBe('cascade');
    expect(parseLiqView('hunt')).toBe('hunt');
    expect(parseLiqView('max-pain')).toBe('max-pain');
    expect(parseLiqHuntPanel('path')).toBe('path');
    expect(parseLiqHuntPanel(null)).toBe('compare');
    expect(parseLiqView(null)).toBe('overview');
    expect(parseLiqCascadeSymbol(null)).toBe('all');
    expect(parseLiqCascadeSymbol('eth')).toBe('ETHUSDT');
    expect(parseLiqChartSymbol('all')).toBe('all');
    expect(parseLiqChartSymbol('eth')).toBe('ETHUSDT');
    expect(parseLiqWindow('12h')).toBe('12h');
    expect(parseLiqWindow('5m')).toBe('24h');
    expect(parseLiqExchange('bybit')).toBe('bybit');
    expect(parseLiqExchange('coinbase')).toBe('all');
    expect(parseLiqSymbol('btc-usd')).toBe('BTCUSDT');
    expect(parseLiqSymbol('')).toBe('BTCUSDT');
  });
});
