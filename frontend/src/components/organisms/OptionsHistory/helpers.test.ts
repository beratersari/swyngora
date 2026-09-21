import { describe, expect, it } from 'vitest';
import { formatHistoryDelta, formatVolPts, historyWindow, toHistoryLine, windowChipLabel } from './helpers';

describe('options history helpers', () => {
  it('formats a signed percent', () => {
    expect(formatHistoryDelta('12.34')).toBe('+12.3%');
    expect(formatHistoryDelta('-3')).toBe('-3.0%');
    expect(formatHistoryDelta(undefined)).toBe('');
  });

  it('picks a window by id', () => {
    expect(historyWindow([{ window: '24h', oi: { changePct: '4' } }], '24h')?.oi?.changePct).toBe('4');
  });

  it('maps OI points inside the selected range', () => {
    const now = Date.parse('2026-03-02T12:00:00Z');
    const line = toHistoryLine(
      [
        { time: '2026-03-01T00:00:00Z', openInterestUsd: '80' },
        { time: '2026-03-02T06:00:00Z', openInterestUsd: '100' },
        { time: '2026-03-02T12:00:00Z', openInterestUsd: '120' },
      ],
      'oi',
      '12h',
      now,
    );
    expect(line).toHaveLength(2);
    expect(line[1]?.value).toBe(120);
  });

  it('maps GEX and flip lines', () => {
    const now = Date.parse('2026-03-02T12:00:00Z');
    const points = [
      {
        time: '2026-03-02T06:00:00Z',
        gammaNetUsd: '400',
        gammaFlip: '102000',
        indexPrice: '100000',
      },
      {
        time: '2026-03-02T12:00:00Z',
        gammaNetUsd: '-200',
        gammaFlip: '98000',
        indexPrice: '101000',
      },
    ];
    expect(toHistoryLine(points, 'gex', '12h', now).map((p) => p.value)).toEqual([400, -200]);
    expect(toHistoryLine(points, 'flip', '12h', now)[1]?.value).toBe(98000);
    expect(toHistoryLine(points, 'index', '12h', now)[1]?.value).toBe(101000);
  });

  it('labels a regime change on the window chip', () => {
    const chip = windowChipLabel({ window: '24h', regime: 'short', regimeWas: 'long' }, 'gex');
    expect(chip.note).toBe('regime');
    expect(chip.to).toBe('short');
  });

  it('maps realized vol and labels the DVOL−RV gap change', () => {
    const now = Date.parse('2026-03-02T12:00:00Z');
    const points = [
      { time: '2026-03-02T06:00:00Z', dvol: '62', realizedVol: '44', volGap: '18' },
      { time: '2026-03-02T12:00:00Z', dvol: '70', realizedVol: '45', volGap: '25' },
    ];
    expect(toHistoryLine(points, 'realized', '12h', now).map((p) => p.value)).toEqual([44, 45]);
    expect(toHistoryLine(points, 'iv', '12h', now)[1]?.value).toBe(70);
    expect(formatVolPts('25')).toBe('+25.0');
    expect(windowChipLabel({ window: '24h', volGap: { change: '5' } }, 'iv').delta).toBe('+5.0');
  });
});
