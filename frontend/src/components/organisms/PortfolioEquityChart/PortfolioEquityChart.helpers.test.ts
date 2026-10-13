import { describe, expect, it } from 'vitest';
import { TickMarkType } from 'lightweight-charts';
import {
  drawdownLine,
  formatEquityTickMark,
  periodChangeLine,
  performanceEquity,
  toEquityLineData,
} from './PortfolioEquityChart.helpers';

describe('toEquityLineData', () => {
  it('dedupes timestamps and keeps latest value', () => {
    const data = toEquityLineData([
      { t: '2026-08-12T12:00:00.000Z', equity: 100 },
      { t: '2026-08-12T12:00:00.000Z', equity: 110 },
      { t: '2026-08-12T13:00:00.000Z', equity: 120 },
    ]);
    expect(data).toHaveLength(2);
    expect(data[0].value).toBe(110);
    expect(data[1].value).toBe(120);
  });

  it('expands a single live point using startEquity/startAt', () => {
    const data = toEquityLineData([{ t: '2026-08-12T13:00:00.000Z', equity: 9990 }], {
      startAt: '2026-08-12T12:00:00.000Z',
      startEquity: 10000,
    });
    expect(data).toHaveLength(2);
    expect(data[0].value).toBe(10000);
    expect(data[1].value).toBe(9990);
    expect(Number(data[0].time)).toBeLessThan(Number(data[1].time));
  });

  it('removes deposits and withdrawals from the line', () => {
    const data = toEquityLineData([
      { t: '2026-08-12T12:00:00.000Z', equity: 10000, externalFlow: 0 },
      { t: '2026-08-12T13:00:00.000Z', equity: 15000, externalFlow: 5000 },
      { t: '2026-08-12T14:00:00.000Z', equity: 12000, externalFlow: 3000 },
    ]);
    expect(data.map((p) => p.value)).toEqual([10000, 10000, 9000]);
    expect(performanceEquity({ equity: 8000, externalFlow: -2000 })).toBe(10000);
  });
});

describe('periodChangeLine', () => {
  it('signs the trading P&L and percent', () => {
    expect(periodChangeLine(20, 0.2, 'USDT')).toBe('+20 USDT (+0.2%)');
    expect(periodChangeLine(0, 0, 'USDT')).toBe('0 USDT (0%)');
    expect(periodChangeLine(-15, null, 'USDT')).toBe('-15 USDT');
  });
});

describe('drawdownLine', () => {
  it('names the largest drop and the times it ran between', () => {
    const line = drawdownLine(
      {
        maxDrawdown: 4000,
        maxDrawdownPct: 33.333,
        drawdownFrom: '2026-08-01T11:00:00.000Z',
        drawdownTo: '2026-08-01T15:00:00.000Z',
      },
      'en-US',
    );
    expect(line.startsWith('4.00K (33.33%) · ')).toBe(true);
    expect(line).toContain('→');
  });

  it('stays empty when equity never fell', () => {
    expect(drawdownLine({ maxDrawdown: 0 })).toBe('');
  });
});

describe('formatEquityTickMark', () => {
  const noon = Math.floor(Date.parse('2026-08-12T12:00:00.000Z') / 1000);
  const afternoon = Math.floor(Date.parse('2026-08-12T18:00:00.000Z') / 1000);

  it('uses time (not day-only) for 1d DayOfMonth ticks so same-day labels differ', () => {
    const a = formatEquityTickMark(noon, TickMarkType.DayOfMonth, 'tr-TR', '1d', 6 * 3600);
    const b = formatEquityTickMark(afternoon, TickMarkType.DayOfMonth, 'tr-TR', '1d', 6 * 3600);
    // Must include a time component; must not be identical day-only labels.
    expect(a).not.toBe(b);
    expect(a).toMatch(/\d/);
    expect(b).toMatch(/\d/);
    // Day-only Turkish form like "12 Ağu" should not be the whole label for 1d.
    expect(a === '12 Ağu' || a === '12 Agu').toBe(false);
  });

  it('keeps day labels for multi-day windows on DayOfMonth marks', () => {
    const label = formatEquityTickMark(
      noon,
      TickMarkType.DayOfMonth,
      'en-US',
      '1w',
      7 * 24 * 3600,
    );
    expect(label.toLowerCase()).toMatch(/aug|12/);
    // Should not force hour for week view day marks
    expect(label).not.toMatch(/:/);
  });
});
