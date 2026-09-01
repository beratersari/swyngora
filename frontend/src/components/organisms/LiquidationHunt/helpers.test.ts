import { describe, expect, it } from 'vitest';
import {
  compareRows,
  coverageTone,
  easeTone,
  formatTarget,
  formatEffect,
  inputSpanText,
  inputTone,
  leanTone,
  defaultHuntWeightDraft,
  huntWeightTotal,
  isDefaultHuntWeightDraft,
  parseHuntPanel,
  parseHuntWeightDraft,
  pathLeverageLabel,
  pathStepTone,
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
    expect(inputSpanText('18m', '1h', 25)).toBe('18m / 1h (25%)');
    expect(inputSpanText('2h', '1h', 50, '2h', true)).toBe('2h old / 1h (50%)');
    expect(formatEffect(4.2)).toBe('+4.2');
    expect(formatEffect(-1.5)).toBe('−1.5');
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

  it('parses the hunt panel and path step tones', () => {
    expect(parseHuntPanel('path')).toBe('path');
    expect(parseHuntPanel('compare')).toBe('compare');
    expect(parseHuntPanel(null)).toBe('compare');
    expect(pathStepTone({ role: 'unreachable' })).toBe('unreachable');
    expect(pathStepTone({ role: 'self' })).toBe('self');
    expect(pathStepTone({ role: 'helped' })).toBe('helped');
    expect(pathStepTone({ role: 'stall' })).toBe('stall');
    expect(pathStepTone({ role: 'missing' })).toBe('missing');
    expect(pathStepTone({ role: 'start' })).toBe('start');
    expect(pathLeverageLabel({ band: { leverage: '125' } })).toBe('125x');
  });

  it('parses custom hunt weights without silent normalize', () => {
    const def = defaultHuntWeightDraft();
    expect(huntWeightTotal(def)).toBe(100);
    expect(isDefaultHuntWeightDraft(def)).toBe(true);
    const custom = parseHuntWeightDraft((k) => (k === 'weightProximity' ? '40' : k === 'weightBook' ? '60' : ''));
    expect(custom).not.toBeNull();
    expect(huntWeightTotal(custom ?? [])).toBe(100);
    expect(custom?.find((r) => r.id === 'trend')?.enabled).toBe(false);
    const partial = parseHuntWeightDraft((k) => (k === 'weightProximity' ? '40' : ''));
    expect(huntWeightTotal(partial ?? [])).toBe(40);
  });
});
