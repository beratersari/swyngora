import { describe, expect, it } from 'vitest';
import { GAUGE_CX, GAUGE_CY, GAUGE_R } from './constants';
import { gaugeArc, gaugePoint } from './helpers';

describe('gaugePoint', () => {
  it('puts 0 on the left, 50 at the top, and 100 on the right', () => {
    const left = gaugePoint(0);
    const top = gaugePoint(50);
    const right = gaugePoint(100);
    expect(left.x).toBeCloseTo(GAUGE_CX - GAUGE_R, 0);
    expect(left.y).toBeCloseTo(GAUGE_CY, 0);
    expect(top.x).toBeCloseTo(GAUGE_CX, 0);
    expect(top.y).toBeCloseTo(GAUGE_CY - GAUGE_R, 0);
    expect(right.x).toBeCloseTo(GAUGE_CX + GAUGE_R, 0);
    expect(right.y).toBeCloseTo(GAUGE_CY, 0);
  });
});

describe('gaugeArc', () => {
  it('sweeps the upper half from the left end to the right end', () => {
    const path = gaugeArc(0, 100);
    expect(path.startsWith(`M ${GAUGE_CX - GAUGE_R}`)).toBe(true);
    expect(path).toContain(`${GAUGE_CX + GAUGE_R}`);
    expect(path).toContain('0 0 0');
  });
});
