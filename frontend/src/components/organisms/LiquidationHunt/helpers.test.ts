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
  huntLeanFromScores,
  huntScoreKeep,
  huntWeightTotal,
  isDefaultHuntWeightDraft,
  parseHuntPanel,
  parseHuntWeightDraft,
  pathLeverageLabel,
  pathStepTone,
  previewHuntMix,
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

  it('does not redistribute a missing factor when previewing a custom mix', () => {
    expect(huntLeanFromScores(70, 50).lean).toBe('up');
    expect(huntScoreKeep(100)).toBeCloseTo(1, 5);
    const venue: HuntVenue = {
      coverage: { score: 100, usable: true },
      openInterestValue: '1',
      upScore: {
        factors: [
          { id: 'proximity', score: 80, status: 'used', requestedPct: 20 },
          { id: 'book', score: 60, status: 'used', requestedPct: 16 },
          { id: 'trend', score: 0, status: 'missing', requestedPct: 20 },
        ],
      },
      downScore: {
        factors: [
          { id: 'proximity', score: 40, status: 'used' },
          { id: 'book', score: 50, status: 'used' },
          { id: 'trend', status: 'missing' },
        ],
      },
    };
    const draft = defaultHuntWeightDraft().map((row) => {
      if (row.id === 'proximity') return { ...row, enabled: true, pct: 40 };
      if (row.id === 'book') return { ...row, enabled: true, pct: 30 };
      if (row.id === 'trend') return { ...row, enabled: true, pct: 30 };
      return { ...row, enabled: false, pct: 0 };
    });
    const preview = previewHuntMix([venue], draft);
    expect(preview).not.toBeNull();
    expect(preview?.appliedUp).toBeCloseTo(65, 0);
    const trend = preview?.upFactors.find((f) => f.id === 'trend');
    expect(trend?.status).toBe('missing');
    expect(trend?.appliedPct).toBe(30);
    expect(trend?.appliedEffect).toBe(0);
    const prox = preview?.upFactors.find((f) => f.id === 'proximity');
    expect(prox?.appliedPct).toBe(40);
    expect(prox?.appliedEffect).toBeCloseTo(12, 0);
    const downProx = preview?.downFactors.find((f) => f.id === 'proximity');
    expect(downProx?.appliedEffect).toBeCloseTo(-4, 0);
    expect(preview?.upLargestChange?.id).toBe('proximity');
    expect(preview?.downLargestChange?.id).toBe('proximity');
    expect(preview?.upLargestChange?.deltaEffect).not.toBe(preview?.downLargestChange?.deltaEffect);
  });

  it('lists which venues used or missed a combined factor', () => {
    const draft = defaultHuntWeightDraft();
    const preview = previewHuntMix(
      [
        {
          exchange: 'binance',
          coverage: { score: 90, usable: true },
          openInterestValue: '10',
          upScore: { factors: [{ id: 'flow', score: 0, status: 'missing' }] },
          downScore: { factors: [{ id: 'flow', score: 0, status: 'missing' }] },
        },
        {
          exchange: 'bybit',
          coverage: { score: 90, usable: true },
          openInterestValue: '10',
          upScore: { factors: [{ id: 'flow', score: 70, status: 'used' }] },
          downScore: { factors: [{ id: 'flow', score: 40, status: 'used' }] },
        },
      ],
      draft,
    );
    const flow = preview?.upFactors.find((f) => f.id === 'flow');
    expect(flow?.status).toBe('used');
    expect(flow?.usedVenues).toEqual(['bybit']);
    expect(flow?.missingVenues).toEqual(['binance']);
    expect(flow?.appliedEffect).toBeCloseTo((70 - 50) * 0.14, 0);
  });
});
