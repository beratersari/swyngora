import { describe, expect, it } from 'vitest';
import {
  compareRows,
  coverageTone,
  easeTone,
  formatTarget,
  inputTone,
  leanTone,
  scoreValue,
  venueLabel,
} from './helpers';
import type { HuntVenue } from './LiquidationHunt.types';

describe('LiquidationHunt helpers', () => {
  it('maps lean and ease tones', () => {
    expect(leanTone('up')).toBe('up');
    expect(leanTone('down')).toBe('down');
    expect(leanTone('even')).toBe('even');
    expect(easeTone('easier')).toBe('easier');
    expect(easeTone('nope')).toBe('mixed');
    expect(scoreValue(140)).toBe(100);
    expect(venueLabel('binance')).toBe('Binance');
    expect(coverageTone('thin')).toBe('thin');
    expect(inputTone('error')).toBe('error');
  });

  it('formats a target with move and builds compare rows', () => {
    expect(formatTarget({ target: { price: '100', movePct: '1.25' } })).toBe('100 (+1.25%)');
    const venue: HuntVenue = {
      upHunt: {
        target: { price: '102', movePct: '2' },
        spot: { notional: '1000', reachable: true },
        estLiquidated: '5000',
        efficiency: '5',
        netWithCascade: '12',
        houseEdge: 'profit',
      },
      downHunt: {
        target: { price: '96', movePct: '-4' },
        spot: { notional: '4000', reachable: true },
        estLiquidated: '2000',
        efficiency: '0.5',
        netWithCascade: '-40',
        houseEdge: 'loss',
      },
    };
    const rows = compareRows(venue, (v) => String(v ?? '—'));
    expect(rows.find((r) => r.id === 'spot')?.upTone).toBe('up');
    expect(rows.find((r) => r.id === 'desk')?.downTone).toBe('loss');
  });
});
